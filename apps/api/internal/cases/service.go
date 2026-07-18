package cases

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/notifications"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

var (
	ErrNotFound       = errors.New("case not found")
	ErrForbidden      = errors.New("forbidden")
	ErrTagLimit       = errors.New("maximum 20 tags per case")
	ErrUBOGateBlocked = errors.New("ubo verification required before closure")
	ErrCaseClosed     = errors.New("case is closed — reopen before making changes")
)

// DefaultOrgID is used in MVP while multi-tenancy is deferred.
var DefaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// AssignmentEngine is a minimal interface so the cases package doesn't import assignment directly.
type AssignmentEngine interface {
	Evaluate(ctx context.Context, input AssignmentInput) (uuid.UUID, error)
}

// AssignmentInput matches assignment.MatchInput to avoid a direct import.
type AssignmentInput struct {
	Priority string
	CaseType string
	SourceID *uuid.UUID
}

// PlaybookApplier is the narrow interface used to trigger playbook auto-apply
// after case creation without importing the playbooks package directly.
type PlaybookApplier interface {
	ApplyToCase(ctx context.Context, caseID uuid.UUID, createdBy uuid.UUID, caseType string, caseCreatedAt time.Time) error
}

// UBOGate is the narrow interface to check UBO verification before closure.
// Avoids a direct import of the ubo package from cases.
type UBOGate interface {
	CaseMayClose(ctx context.Context, caseID string) error
}

type Service struct {
	repo            *Repository
	sqlDB           *sql.DB // needed for bulk-op transactions
	emitter         timeline.Emitter
	notifSvc        *notifications.Service
	assignEng       AssignmentEngine
	playbookApplier PlaybookApplier
	uboGate         UBOGate
}

func NewService(repo *Repository, sqlDB *sql.DB, emitter timeline.Emitter) *Service {
	repo.WithDB(sqlDB)
	return &Service{repo: repo, sqlDB: sqlDB, emitter: emitter}
}

func (s *Service) WithNotifications(notifSvc *notifications.Service) *Service {
	s.notifSvc = notifSvc
	return s
}

func (s *Service) WithAssignmentEngine(e AssignmentEngine) *Service {
	s.assignEng = e
	return s
}

func (s *Service) WithPlaybookApplier(pa PlaybookApplier) *Service {
	s.playbookApplier = pa
	return s
}

func (s *Service) WithUBO(gate UBOGate) *Service {
	s.uboGate = gate
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
	} else if s.assignEng != nil {
		if autoUser, err := s.assignEng.Evaluate(ctx, AssignmentInput{
			Priority: string(req.Priority),
			CaseType: req.CaseType,
		}); err == nil && autoUser != uuid.Nil {
			assignedTo = uuid.NullUUID{UUID: autoUser, Valid: true}
		}
	}

	creatorID := claims.UserID

	c, err := s.repo.Create(ctx, db.CreateCaseParams{
		Title:          req.Title,
		Description:    req.Description,
		Status:         db.CaseStatusOpen,
		Priority:       req.Priority,
		AssignedTo:     assignedTo,
		CreatedBy:      creatorID,
		OrganizationID: DefaultOrgID,
		DueDate:        nullTimeFromPtr(req.DueDate),
		SlaBreachAt:    sql.NullTime{Time: slaBreachAt(req.Priority, time.Now()), Valid: true},
		Tags:           tags,
	})
	if err != nil {
		return nil, err
	}

	if req.CaseType != "" {
		s.sqlDB.ExecContext(ctx, `UPDATE cases SET case_type = $1 WHERE id = $2`, req.CaseType, c.ID) //nolint:errcheck
	}

	caseID, caseTitle, casePriority, caseNumber, caseCreatedAt := c.ID, c.Title, c.Priority, c.CaseNumber, c.CreatedAt
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), caseID, creatorID, "case_created", map[string]interface{}{ //nolint:errcheck
			"title":    caseTitle,
			"priority": casePriority,
		})
	})
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  db.TimelineEventTypeCaseCreated,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &creatorID,
			OccurredAt: caseCreatedAt,
			Payload: map[string]any{
				"case_number": caseNumber,
				"title":       caseTitle,
				"priority":    string(casePriority),
			},
		})
	})

	if s.playbookApplier != nil && req.CaseType != "" {
		caseType := req.CaseType
		worker.TrySubmit(func() {
			s.playbookApplier.ApplyToCase(context.Background(), caseID, creatorID, caseType, caseCreatedAt) //nolint:errcheck
		})
	}

	resp := s.toCaseResponse(c, "", claims.Email, nil, nil, nil)
	resp.CaseType = req.CaseType
	return resp, nil
}

// CreateAuto creates a case on behalf of the system (alert automation, no user JWT).
// createdBy should be the org admin UUID or the system sentinel UUID.
func (s *Service) CreateAuto(ctx context.Context, createdBy uuid.UUID, req CreateCaseRequest) (*CaseResponse, error) {
	tags := req.Tags
	if tags == nil {
		tags = []string{}
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
	} else if s.assignEng != nil {
		if autoUser, err := s.assignEng.Evaluate(ctx, AssignmentInput{Priority: string(req.Priority), CaseType: req.CaseType}); err == nil && autoUser != uuid.Nil {
			assignedTo = uuid.NullUUID{UUID: autoUser, Valid: true}
		}
	}

	c, err := s.repo.Create(ctx, db.CreateCaseParams{
		Title:          req.Title,
		Description:    req.Description,
		Status:         db.CaseStatusOpen,
		Priority:       req.Priority,
		AssignedTo:     assignedTo,
		CreatedBy:      createdBy,
		OrganizationID: DefaultOrgID,
		DueDate:        nullTimeFromPtr(req.DueDate),
		SlaBreachAt:    sql.NullTime{},
		Tags:           tags,
	})
	if err != nil {
		return nil, err
	}

	if req.CaseType != "" {
		s.sqlDB.ExecContext(ctx, `UPDATE cases SET case_type = $1 WHERE id = $2`, req.CaseType, c.ID) //nolint:errcheck
	}

	caseID, caseTitle, casePriority, caseNumber, caseCreatedAt := c.ID, c.Title, c.Priority, c.CaseNumber, c.CreatedAt
	actor := createdBy
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), caseID, actor, "case_created", map[string]interface{}{ //nolint:errcheck
			"title":    caseTitle,
			"priority": casePriority,
			"source":   "alert_automation",
		})
	})
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     caseID,
			EventType:  db.TimelineEventTypeCaseCreated,
			Source:     db.TimelineEventSourceSystem,
			ActorID:    &actor,
			OccurredAt: caseCreatedAt,
			Payload: map[string]any{
				"case_number": caseNumber,
				"title":       caseTitle,
				"priority":    string(casePriority),
				"source":      "alert_automation",
			},
		})
	})

	if s.playbookApplier != nil && req.CaseType != "" {
		caseType := req.CaseType
		worker.TrySubmit(func() {
			s.playbookApplier.ApplyToCase(context.Background(), caseID, actor, caseType, caseCreatedAt) //nolint:errcheck
		})
	}

	resp := s.toCaseResponse(c, "", "", nil, nil, nil)
	resp.CaseType = req.CaseType
	return resp, nil
}

func (s *Service) Get(ctx context.Context, id, orgID uuid.UUID) (*CaseResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	// IDOR guard: case must belong to the caller's org (RBI C6)
	if row.OrganizationID != orgID {
		return nil, ErrNotFound
	}
	watchers, _ := s.repo.GetWatchers(ctx, id)
	return s.toResponseFromGetRow(row, watchers), nil
}

func (s *Service) List(ctx context.Context, f ListFilters) ([]CaseResponse, ListMeta, error) {
	if f.ActiveOnly {
		return s.listActive(ctx, f)
	}
	if len(f.Statuses) > 0 {
		return s.listByStatuses(ctx, f)
	}

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

func (s *Service) listActive(ctx context.Context, f ListFilters) ([]CaseResponse, ListMeta, error) {
	rows, err := s.repo.ListActive(ctx, f, DefaultOrgID)
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

	total, _ := s.repo.CountActive(ctx, f, DefaultOrgID)

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

func (s *Service) listByStatuses(ctx context.Context, f ListFilters) ([]CaseResponse, ListMeta, error) {
	rows, err := s.repo.ListByStatuses(ctx, f, DefaultOrgID)
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

	total, _ := s.repo.CountByStatuses(ctx, f, DefaultOrgID)

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

	if existing.Status == db.CaseStatusClosed {
		return nil, ErrCaseClosed
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
	var slaParams *db.UpdateCaseStatusParams
	if req.Priority != nil {
		p.Priority = db.NullCasePriority{CasePriority: *req.Priority, Valid: true}
		if existing.Status != db.CaseStatusDraft {
			newSLA := slaBreachAt(*req.Priority, time.Now())
			slaParams = &db.UpdateCaseStatusParams{
				ID:          id,
				Status:      existing.Status,
				SlaBreachAt: sql.NullTime{Time: newSLA, Valid: true},
				SlaBreached: sql.NullBool{Bool: false, Valid: true},
			}
		}
	}

	c, err := s.repo.Update(ctx, p)
	if err != nil {
		return nil, err
	}

	if slaParams != nil {
		if updated, err := s.repo.UpdateStatus(ctx, *slaParams); err == nil {
			c = updated
		}
	}

	updateID := id
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), updateID, actorID, "case_updated", map[string]interface{}{}) //nolint:errcheck
	})

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

	// UBO closure gate: block pending_review / closed if UBO records are unverified.
	if req.Status == db.CaseStatusPendingReview || req.Status == db.CaseStatusClosed {
		if s.uboGate != nil {
			if err := s.uboGate.CaseMayClose(ctx, id.String()); err != nil {
				return nil, fmt.Errorf("%w: %s", ErrUBOGateBlocked, err.Error())
			}
		}
	}

	// Maker-checker: prevent the same person from submitting and approving.
	if req.Status == db.CaseStatusClosed && existing.Status == db.CaseStatusPendingReview {
		submitterID, err := s.repo.GetLastPendingReviewActor(ctx, id)
		if err == nil && submitterID == actorID {
			return nil, ErrSamePersonApproval
		}
	}

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

	fromStatus, toStatus, reason := string(existing.Status), string(req.Status), req.Reason
	statusCaseID := id
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), statusCaseID, actorID, "status_changed", map[string]interface{}{ //nolint:errcheck
			"from":   fromStatus,
			"to":     toStatus,
			"reason": reason,
		})
	})
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     statusCaseID,
			EventType:  db.TimelineEventTypeCaseStatusChanged,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"from":   fromStatus,
				"to":     toStatus,
				"reason": reason,
			},
		})
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

	// Notify all managers/admins when a case needs review — they have no other signal
	if s.notifSvc != nil && req.Status == db.CaseStatusPendingReview {
		caseIDCopy := id
		caseNumber := existing.CaseNumber
		actorEmail := claims.Email
		notifSvc := s.notifSvc
		repo := s.repo
		worker.TrySubmit(func() {
			managers, err := repo.ListManagers(context.Background())
			if err != nil {
				return
			}
			for _, mgr := range managers {
				if mgr.ID == actorID {
					continue // don't self-notify if a manager submitted it
				}
				notifSvc.NotifyAsync(context.Background(), notifications.NotifyInput{
					RecipientID: mgr.ID,
					Type:        db.NotificationTypeCaseStatusChanged,
					Priority:    db.NotificationPriorityHigh,
					Title:       "Case awaiting your review",
					Message:     "Case " + caseNumber + " has been submitted for review by " + actorEmail,
					LinkURL:     "/cases/" + caseIDCopy.String(),
					CaseID:      &caseIDCopy,
				})
			}
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

	assignCaseID, assignTo, assignNote := id, req.UserID.String(), req.Note
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), assignCaseID, actorID, "assigned", map[string]interface{}{ //nolint:errcheck
			"to":   assignTo,
			"note": assignNote,
		})
	})
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     assignCaseID,
			EventType:  db.TimelineEventTypeCaseAssigned,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"assigned_to": assignTo,
				"note":        assignNote,
			},
		})
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
	tagCaseID, addTags, removeTags := id, req.Add, req.Remove
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), tagCaseID, actorID, "tags_updated", map[string]interface{}{ //nolint:errcheck
			"add":    addTags,
			"remove": removeTags,
		})
	})
	worker.TrySubmit(func() {
		s.emitter.Emit(context.Background(), timeline.Event{ //nolint:errcheck
			CaseID:     tagCaseID,
			EventType:  db.TimelineEventTypeCaseTagged,
			Source:     db.TimelineEventSourceUser,
			ActorID:    &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"add":    addTags,
				"remove": removeTags,
			},
		})
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
	delCaseID := id
	worker.TrySubmit(func() {
		s.repo.CreateActivity(context.Background(), delCaseID, actorID, "case_deleted", map[string]interface{}{}) //nolint:errcheck
	})
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
	if row.RiskScore.Valid {
		rs := row.RiskScore.String
		resp.RiskScore = &rs
	}
	if row.RiskLevel.Valid {
		rl := row.RiskLevel.String
		resp.RiskLevel = &rl
	}
	resp.OpenTasksCount = int(row.OpenTasksCount)
	resp.DocumentsCount = int(row.DocumentsCount)
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
	if row.RiskScore.Valid {
		s := row.RiskScore.String
		r.RiskScore = &s
	}
	if row.RiskLevel.Valid {
		l := row.RiskLevel.String
		r.RiskLevel = &l
	}
	return r
}

// BulkStatus updates the status for multiple cases inside a single transaction.
// All updates succeed or all are rolled back — no partial commits.
func (s *Service) BulkStatus(ctx context.Context, claims *auth.Claims, ids []uuid.UUID, status db.CaseStatus) (updated int, failed []uuid.UUID) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, ids
	}
	txq := db.New(tx)

	for _, id := range ids {
		if _, err := txq.UpdateCaseStatus(ctx, db.UpdateCaseStatusParams{
			ID:     id,
			Status: status,
		}); err != nil {
			failed = append(failed, id)
		} else {
			updated++
		}
	}

	if len(failed) > 0 {
		_ = tx.Rollback()
		return 0, ids // all-or-nothing: roll back on any failure
	}
	if err := tx.Commit(); err != nil {
		return 0, ids
	}
	return updated, nil
}

// BulkAssign reassigns multiple cases to a single assignee inside a single transaction.
// Also writes case_assignments history entries for each case (chain-of-custody audit trail).
func (s *Service) BulkAssign(ctx context.Context, claims *auth.Claims, ids []uuid.UUID, assigneeID uuid.UUID) (updated int, failed []uuid.UUID) {
	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, ids
	}
	txq := db.New(tx)

	for _, id := range ids {
		if _, err := txq.UpdateCaseAssignment(ctx, db.UpdateCaseAssignmentParams{
			ID:         id,
			AssignedTo: uuid.NullUUID{UUID: assigneeID, Valid: true},
		}); err != nil {
			failed = append(failed, id)
			continue
		}
		// Record assignment history — closes prior open assignment and opens a new one.
		// Errors here are non-fatal (history is best-effort inside the same tx).
		_ = txq.CloseLastAssignment(ctx, id)
		_, _ = txq.CreateCaseAssignment(ctx, db.CreateCaseAssignmentParams{
			CaseID:     id,
			AssignedTo: assigneeID,
			AssignedBy: claims.UserID,
		})
		updated++
	}

	if len(failed) > 0 {
		_ = tx.Rollback()
		return 0, ids
	}
	if err := tx.Commit(); err != nil {
		return 0, ids
	}
	return updated, nil
}

// BulkDelete soft-deletes multiple cases inside a single transaction.
// Analysts may only delete cases assigned to them; managers and above are unrestricted.
func (s *Service) BulkDelete(ctx context.Context, claims *auth.Claims, ids []uuid.UUID) (deleted int, failed []uuid.UUID) {
	isPrivileged := claims.Role == db.UserRoleManager ||
		claims.Role == db.UserRoleAdmin ||
		claims.Role == db.UserRoleSuperAdmin

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, ids
	}
	txq := db.New(tx)

	for _, id := range ids {
		if !isPrivileged {
			row, err := txq.GetCaseByID(ctx, id)
			if err != nil || row.AssignedTo.UUID != claims.UserID {
				failed = append(failed, id)
				continue
			}
		}
		if _, err := txq.SoftDeleteCase(ctx, id); err != nil {
			failed = append(failed, id)
		} else {
			deleted++
		}
	}

	if len(failed) > 0 {
		_ = tx.Rollback()
		return 0, ids
	}
	if err := tx.Commit(); err != nil {
		return 0, ids
	}
	return deleted, nil
}

// CompletionSummary holds per-tab content counts for the case tab strip.
// Each tab has a total count + a "pending action" count so the UI can show
// green (done) vs amber (action needed) vs no dot (empty).
type CompletionSummary struct {
	Documents           int  `json:"documents"`
	DocumentsProcessing int  `json:"documents_processing"` // still pending/processing
	Tasks               int  `json:"tasks"`
	TasksOpen           int  `json:"tasks_open"`           // todo or in_progress
	Notes               int  `json:"notes"`
	Comms               int  `json:"comms"`
	InsightsAvailable   bool `json:"insights_available"`
	Transactions        int  `json:"transactions"`
	Findings            int  `json:"findings"`
	FindingsApproved    int  `json:"findings_approved"`
	FindingsPending     int  `json:"findings_pending"`     // draft or under_review
	STRExists           bool `json:"str_exists"`
	STRFiled            bool `json:"str_filed"`
	NarrativeExists     bool `json:"narrative_exists"`
	TimelineEvents      int  `json:"timeline_events"`
}

func (s *Service) GetCompletion(ctx context.Context, caseID uuid.UUID) (*CompletionSummary, error) {
	const q = `
SELECT
  (SELECT COUNT(*) FROM documents WHERE case_id = $1)::int                                                            AS documents,
  (SELECT COUNT(*) FROM documents WHERE case_id = $1 AND processing_status IN ('pending','processing'))::int          AS documents_processing,
  (SELECT COUNT(*) FROM tasks     WHERE case_id = $1 AND deleted_at IS NULL)::int                                     AS tasks,
  (SELECT COUNT(*) FROM tasks     WHERE case_id = $1 AND deleted_at IS NULL AND status IN ('todo','in_progress'))::int AS tasks_open,
  (SELECT COUNT(*) FROM notes     WHERE case_id = $1)::int                                                            AS notes,
  (SELECT COUNT(*) FROM communication_logs WHERE case_id = $1)::int                                                   AS comms,
  EXISTS(SELECT 1 FROM case_ai_insights WHERE case_id = $1 AND risk_score IS NOT NULL)                             AS insights_available,
  (SELECT COUNT(*) FROM case_transactions WHERE case_id = $1)::int                                                    AS transactions,
  (SELECT COUNT(*) FROM case_findings WHERE case_id = $1)::int                                                        AS findings,
  (SELECT COUNT(*) FROM case_findings WHERE case_id = $1 AND status = 'approved')::int                               AS findings_approved,
  (SELECT COUNT(*) FROM case_findings WHERE case_id = $1 AND status IN ('draft','under_review'))::int                AS findings_pending,
  EXISTS(SELECT 1 FROM str_filings WHERE case_id = $1)                                                               AS str_exists,
  EXISTS(SELECT 1 FROM str_filings WHERE case_id = $1 AND status = 'filed')                                          AS str_filed,
  EXISTS(SELECT 1 FROM case_narratives WHERE case_id = $1)                                                            AS narrative_exists,
  (SELECT COUNT(*) FROM timeline_events WHERE case_id = $1)::int                                                      AS timeline_events
`
	var c CompletionSummary
	row := s.sqlDB.QueryRowContext(ctx, q, caseID)
	if err := row.Scan(
		&c.Documents, &c.DocumentsProcessing,
		&c.Tasks, &c.TasksOpen,
		&c.Notes, &c.Comms,
		&c.InsightsAvailable, &c.Transactions,
		&c.Findings, &c.FindingsApproved, &c.FindingsPending,
		&c.STRExists, &c.STRFiled,
		&c.NarrativeExists, &c.TimelineEvents,
	); err != nil {
		return nil, err
	}
	return &c, nil
}
