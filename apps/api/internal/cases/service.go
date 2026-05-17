package cases

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/notifications"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

var (
	ErrNotFound      = errors.New("case not found")
	ErrForbidden     = errors.New("forbidden")
	ErrTagLimit      = errors.New("maximum 20 tags per case")
)

// DefaultOrgID is used in MVP while multi-tenancy is deferred.
var DefaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Service struct {
	repo     *Repository
	emitter  timeline.Emitter
	notifSvc *notifications.Service
}

func NewService(repo *Repository, emitter timeline.Emitter) *Service {
	return &Service{repo: repo, emitter: emitter}
}

func (s *Service) WithNotifications(notifSvc *notifications.Service) *Service {
	s.notifSvc = notifSvc
	return s
}

func (s *Service) Create(ctx context.Context, claims *auth.Claims, req CreateCaseRequest) (*CaseResponse, error) {
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	if len(tags) > 20 {
		return nil, ErrTagLimit
	}
	for i, t := range tags {
		tags[i] = strings.ToLower(strings.TrimSpace(t))
		if _, err := s.repo.UpsertTag(ctx, tags[i]); err != nil {
			return nil, err
		}
	}

	assignedTo := uuid.NullUUID{}
	if req.AssignedTo != nil {
		assignedTo = uuid.NullUUID{UUID: *req.AssignedTo, Valid: true}
	}

	creatorID := claims.UserID

	c, err := s.repo.Create(ctx, db.CreateCaseParams{
		Title:          req.Title,
		Description:    req.Description,
		Status:         db.CaseStatusDraft,
		Priority:       req.Priority,
		AssignedTo:     assignedTo,
		CreatedBy:      creatorID,
		OrganizationID: DefaultOrgID,
		DueDate:        nullTimeFromPtr(req.DueDate),
		SlaBreachAt:    sql.NullTime{},
		Tags:           tags,
	})
	if err != nil {
		return nil, err
	}

	go s.repo.CreateActivity(context.Background(), c.ID, creatorID, "case_created", map[string]interface{}{
		"title":    c.Title,
		"priority": c.Priority,
	})
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:    c.ID,
		EventType: db.TimelineEventTypeCaseCreated,
		Source:    db.TimelineEventSourceUser,
		ActorID:   &creatorID,
		OccurredAt: c.CreatedAt,
		Payload: map[string]any{
			"case_number": c.CaseNumber,
			"title":       c.Title,
			"priority":    string(c.Priority),
		},
	})

	return s.toCaseResponse(c, "", claims.Email, nil, nil, nil), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*CaseResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	watchers, _ := s.repo.GetWatchers(ctx, id)
	return s.toResponseFromGetRow(row, watchers), nil
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]CaseResponse, ListMeta, error) {
	rows, err := s.repo.List(ctx, f, DefaultOrgID)
	if err != nil {
		return nil, ListMeta{}, err
	}

	limit := int(f.Limit)
	if limit <= 0 {
		limit = 20
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.Count(ctx, f, DefaultOrgID)

	var nextCursor *uuid.UUID
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		nextCursor = &last
	}

	out := make([]CaseResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponseFromListRow(r))
	}

	return out, ListMeta{NextCursor: nextCursor, HasMore: hasMore, TotalCount: total}, nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, id uuid.UUID, req UpdateCaseRequest) (*CaseResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	actorID := claims.UserID
	actorRole := claims.Role
	if actorRole != db.UserRoleManager && actorRole != db.UserRoleAdmin && actorRole != db.UserRoleSuperAdmin {
		if existing.AssignedTo.UUID != actorID {
			return nil, ErrForbidden
		}
	}

	p := db.UpdateCaseParams{ID: id}
	if req.Title != nil {
		p.Title = sql.NullString{String: *req.Title, Valid: true}
	}
	if req.Description != nil {
		p.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.DueDate != nil {
		p.DueDate = sql.NullTime{Time: *req.DueDate, Valid: true}
	}
	if req.Priority != nil {
		p.Priority = db.NullCasePriority{CasePriority: *req.Priority, Valid: true}
		if existing.Status != db.CaseStatusDraft {
			newSLA := slaBreachAt(*req.Priority, time.Now())
			go s.repo.UpdateStatus(context.Background(), db.UpdateCaseStatusParams{
				ID:          id,
				Status:      existing.Status,
				SlaBreachAt: sql.NullTime{Time: newSLA, Valid: true},
				SlaBreached: sql.NullBool{Bool: false, Valid: true},
			})
		}
	}

	c, err := s.repo.Update(ctx, p)
	if err != nil {
		return nil, err
	}

	go s.repo.CreateActivity(context.Background(), id, actorID, "case_updated", map[string]interface{}{})

	return s.toCaseResponse(c, existing.CreatedByName, existing.CreatedByEmail, nil, nil, nil), nil
}

func (s *Service) ChangeStatus(ctx context.Context, claims *auth.Claims, id uuid.UUID, req StatusRequest) (*CaseResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	actorRole := claims.Role
	if err := validateTransition(existing.Status, req.Status, actorRole, req.Reason); err != nil {
		return nil, err
	}

	actorID := claims.UserID

	p := db.UpdateCaseStatusParams{ID: id, Status: req.Status}
	if req.Status == db.CaseStatusOpen {
		breach := slaBreachAt(existing.Priority, time.Now())
		p.SlaBreachAt = sql.NullTime{Time: breach, Valid: true}
		p.SlaBreached = sql.NullBool{Bool: false, Valid: true}
	}

	c, err := s.repo.UpdateStatus(ctx, p)
	if err != nil {
		return nil, err
	}

	go s.repo.CreateActivity(context.Background(), id, actorID, "status_changed", map[string]interface{}{
		"from":   string(existing.Status),
		"to":     string(req.Status),
		"reason": req.Reason,
	})
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:    id,
		EventType: db.TimelineEventTypeCaseStatusChanged,
		Source:    db.TimelineEventSourceUser,
		ActorID:   &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"from":   string(existing.Status),
			"to":     string(req.Status),
			"reason": req.Reason,
		},
	})

	// Notify assigned investigator of status change
	if s.notifSvc != nil && existing.AssignedTo.Valid && existing.AssignedTo.UUID != claims.UserID {
		assigneeID := existing.AssignedTo.UUID
		s.notifSvc.NotifyAsync(context.Background(), notifications.NotifyInput{
			RecipientID: assigneeID,
			Type:        db.NotificationTypeCaseStatusChanged,
			Priority:    db.NotificationPriorityMedium,
			Title:       "Case status updated",
			Message:     "Case " + existing.CaseNumber + " moved to " + string(req.Status),
			LinkURL:     "/cases/" + id.String(),
			CaseID:      &id,
		})
	}

	return s.toCaseResponse(c, existing.CreatedByName, existing.CreatedByEmail, nil, nil, nil), nil
}

func (s *Service) Assign(ctx context.Context, claims *auth.Claims, id uuid.UUID, req AssignRequest) (*CaseResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	actorRole := claims.Role
	if actorRole != db.UserRoleManager && actorRole != db.UserRoleAdmin && actorRole != db.UserRoleSuperAdmin {
		return nil, ErrForbiddenTransition
	}

	actorID := claims.UserID
	if err := s.repo.CreateAssignment(ctx, id, req.UserID, actorID); err != nil {
		return nil, err
	}

	c, err := s.repo.UpdateAssignment(ctx, db.UpdateCaseAssignmentParams{
		ID:         id,
		AssignedTo: uuid.NullUUID{UUID: req.UserID, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	go s.repo.CreateActivity(context.Background(), id, actorID, "assigned", map[string]interface{}{
		"to":   req.UserID.String(),
		"note": req.Note,
	})
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:    id,
		EventType: db.TimelineEventTypeCaseAssigned,
		Source:    db.TimelineEventSourceUser,
		ActorID:   &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"assigned_to": req.UserID.String(),
			"note":        req.Note,
		},
	})

	if s.notifSvc != nil {
		s.notifSvc.NotifyAsync(context.Background(), notifications.NotifyInput{
			RecipientID: req.UserID,
			Type:        db.NotificationTypeCaseAssigned,
			Priority:    db.NotificationPriorityHigh,
			Title:       "Case assigned to you",
			Message:     "You have been assigned as investigator on case " + c.CaseNumber,
			LinkURL:     "/cases/" + id.String(),
			CaseID:      &id,
		})
	}

	return s.toCaseResponse(c, existing.CreatedByName, existing.CreatedByEmail, nil, nil, nil), nil
}

func (s *Service) UpdateTags(ctx context.Context, claims *auth.Claims, id uuid.UUID, req TagsRequest) (*CaseResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	tagSet := make(map[string]struct{}, len(existing.Tags))
	for _, t := range existing.Tags {
		tagSet[t] = struct{}{}
	}
	for _, t := range req.Remove {
		delete(tagSet, strings.ToLower(t))
	}
	for _, t := range req.Add {
		lower := strings.ToLower(strings.TrimSpace(t))
		tagSet[lower] = struct{}{}
		s.repo.UpsertTag(ctx, lower) //nolint:errcheck
	}
	if len(tagSet) > 20 {
		return nil, ErrTagLimit
	}

	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}

	c, err := s.repo.UpdateTags(ctx, db.UpdateCaseTagsParams{ID: id, Tags: tags})
	if err != nil {
		return nil, err
	}

	actorID := claims.UserID
	go s.repo.CreateActivity(context.Background(), id, actorID, "tags_updated", map[string]interface{}{
		"add":    req.Add,
		"remove": req.Remove,
	})
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:    id,
		EventType: db.TimelineEventTypeCaseTagged,
		Source:    db.TimelineEventSourceUser,
		ActorID:   &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"add":    req.Add,
			"remove": req.Remove,
		},
	})

	return s.toCaseResponse(c, existing.CreatedByName, existing.CreatedByEmail, nil, nil, nil), nil
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, id uuid.UUID) error {
	actorRole := claims.Role
	if actorRole != db.UserRoleAdmin && actorRole != db.UserRoleSuperAdmin {
		return ErrForbidden
	}
	_, err := s.repo.SoftDelete(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	actorID := claims.UserID
	go s.repo.CreateActivity(context.Background(), id, actorID, "case_deleted", map[string]interface{}{})
	return nil
}

func (s *Service) ListActivities(ctx context.Context, caseID uuid.UUID, cursor *uuid.UUID, limit int32) ([]ActivityResponse, bool, *uuid.UUID, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.repo.ListActivities(ctx, caseID, cursor, limit)
	if err != nil {
		return nil, false, nil, err
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	var nextCursor *uuid.UUID
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1].ID
		nextCursor = &last
	}

	out := make([]ActivityResponse, 0, len(rows))
	for _, r := range rows {
		var payload map[string]interface{}
		_ = json.Unmarshal(r.Payload, &payload)
		out = append(out, ActivityResponse{
			ID:        r.ID,
			EventType: r.EventType,
			Actor:     UserRef{ID: r.ActorID, FullName: r.ActorName, Email: r.ActorEmail},
			Payload:   payload,
			CreatedAt: r.CreatedAt,
		})
	}
	return out, hasMore, nextCursor, nil
}

// ── helpers ───────────────────────────────────────────────────────

func (s *Service) toCaseResponse(c db.Case, creatorName, creatorEmail string, assigneeName, assigneeEmail *string, assigneeRole *string) *CaseResponse {
	r := &CaseResponse{
		ID:             c.ID,
		CaseNumber:     c.CaseNumber,
		Title:          c.Title,
		Description:    c.Description,
		Status:         c.Status,
		Priority:       c.Priority,
		OrganizationID: c.OrganizationID,
		SLABreached:    c.SlaBreached,
		Tags:           c.Tags,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		CreatedBy:      UserRef{ID: c.CreatedBy, FullName: creatorName, Email: creatorEmail},
	}
	if c.SlaBreachAt.Valid {
		r.SLABreachAt = &c.SlaBreachAt.Time
	}
	if c.DueDate.Valid {
		r.DueDate = &c.DueDate.Time
	}
	if c.AssignedTo.Valid {
		ref := &UserRef{ID: c.AssignedTo.UUID}
		if assigneeName != nil {
			ref.FullName = *assigneeName
		}
		if assigneeEmail != nil {
			ref.Email = *assigneeEmail
		}
		if assigneeRole != nil {
			ref.Role = *assigneeRole
		}
		r.AssignedTo = ref
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	if r.Watchers == nil {
		r.Watchers = []UserRef{}
	}
	return r
}

func (s *Service) toResponseFromGetRow(row db.GetCaseByIDRow, watchers []db.GetCaseWatchersRow) *CaseResponse {
	var assigneeName, assigneeEmail, assigneeRole *string
	if row.AssignedToName.Valid {
		assigneeName = &row.AssignedToName.String
	}
	if row.AssignedToEmail.Valid {
		assigneeEmail = &row.AssignedToEmail.String
	}
	if row.AssignedToRole.Valid {
		s := string(row.AssignedToRole.UserRole)
		assigneeRole = &s
	}

	c := db.Case{
		ID: row.ID, CaseNumber: row.CaseNumber, Title: row.Title,
		Description: row.Description, Status: row.Status, Priority: row.Priority,
		AssignedTo: row.AssignedTo, CreatedBy: row.CreatedBy, OrganizationID: row.OrganizationID,
		DueDate: row.DueDate, SlaBreachAt: row.SlaBreachAt, SlaBreached: row.SlaBreached,
		Tags: row.Tags, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	resp := s.toCaseResponse(c, row.CreatedByName, row.CreatedByEmail, assigneeName, assigneeEmail, assigneeRole)

	watcherRefs := make([]UserRef, 0, len(watchers))
	for _, w := range watchers {
		watcherRefs = append(watcherRefs, UserRef{ID: w.ID, FullName: w.FullName, Email: w.Email})
	}
	resp.Watchers = watcherRefs
	return resp
}

func (s *Service) toResponseFromListRow(row db.ListCasesRow) CaseResponse {
	r := CaseResponse{
		ID:             row.ID,
		CaseNumber:     row.CaseNumber,
		Title:          row.Title,
		Description:    row.Description,
		Status:         row.Status,
		Priority:       row.Priority,
		OrganizationID: row.OrganizationID,
		SLABreached:    row.SlaBreached,
		Tags:           row.Tags,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		CreatedBy:      UserRef{ID: row.CreatedBy, FullName: row.CreatedByName},
	}
	if row.SlaBreachAt.Valid {
		r.SLABreachAt = &row.SlaBreachAt.Time
	}
	if row.DueDate.Valid {
		r.DueDate = &row.DueDate.Time
	}
	if row.AssignedTo.Valid {
		ref := &UserRef{ID: row.AssignedTo.UUID}
		if row.AssignedToName.Valid {
			ref.FullName = row.AssignedToName.String
		}
		r.AssignedTo = ref
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	return r
}
