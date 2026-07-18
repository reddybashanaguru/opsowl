package tasks

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/notifications"
	"github.com/reddybashanaguru/opsowl/internal/playbooks"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

type taskCursor struct {
	SortOrder int32     `json:"so"`
	CreatedAt time.Time `json:"ca"`
	ID        uuid.UUID `json:"id"`
}

func encodeCursor(so int32, ca time.Time, id uuid.UUID) string {
	b, _ := json.Marshal(taskCursor{SortOrder: so, CreatedAt: ca, ID: id})
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeCursor decodes an opaque pagination cursor into its composite fields.
// Exported for use in tests.
func DecodeCursor(s string) (sortOrder int32, createdAt time.Time, id uuid.UUID, err error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return
	}
	var c taskCursor
	if err = json.Unmarshal(b, &c); err != nil {
		return
	}
	return c.SortOrder, c.CreatedAt, c.ID, nil
}

type Service struct {
	repo     *Repository
	sqlDB    *sql.DB // for all-tasks view (manager+)
	emitter  timeline.Emitter
	notifSvc *notifications.Service
}

func NewService(repo *Repository, emitter timeline.Emitter) *Service {
	return &Service{repo: repo, emitter: emitter}
}

func (s *Service) WithDB(sqlDB *sql.DB) *Service {
	s.sqlDB = sqlDB
	return s
}

func (s *Service) WithNotifications(notifSvc *notifications.Service) *Service {
	s.notifSvc = notifSvc
	return s
}

func (s *Service) Create(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateTaskRequest) (*TaskResponse, error) {
	if req.Title == "" {
		return nil, ErrTitleRequired
	}

	if s.sqlDB != nil {
		var caseStatus string
		if err := s.sqlDB.QueryRowContext(ctx, `SELECT status FROM cases WHERE id = $1`, caseID).Scan(&caseStatus); err == nil && caseStatus == "closed" {
			return nil, ErrCaseClosed
		}
	}

	priority := req.Priority
	if priority == "" {
		priority = db.TaskPriorityMedium
	}

	now := time.Now()
	isOverdue := req.DueDate != nil && req.DueDate.Before(now)

	t, err := s.repo.Create(ctx, db.CreateTaskParams{
		CaseID:          caseID,
		CreatedBy:       claims.UserID,
		AssignedTo:      nullUUID(req.AssignedTo),
		TimelineEventID: nullUUID(req.TimelineEventID),
		Title:           req.Title,
		Description:     sql.NullString{String: req.Description, Valid: req.Description != ""},
		Priority:        priority,
		DueDate:         nullTime(req.DueDate),
		IsAiSuggested:   req.IsAISuggested,
	})
	if err != nil {
		return nil, err
	}

	if isOverdue {
		// Sync is_overdue immediately for past-due tasks
		if updated, err2 := s.repo.Update(ctx, db.UpdateTaskParams{
			ID:        t.ID,
			IsOverdue: sql.NullBool{Bool: true, Valid: true},
		}); err2 == nil {
			t = updated
		}
	}

	actorID := claims.UserID
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:    caseID,
		EventType: db.TimelineEventTypeTaskCreated,
		Source:    db.TimelineEventSourceUser,
		ActorID:   &actorID,
		OccurredAt: t.CreatedAt,
		Payload: map[string]any{
			"task_id":  t.ID.String(),
			"title":    t.Title,
			"priority": string(t.Priority),
		},
	})

	// Notify assignee if different from creator
	if s.notifSvc != nil && t.AssignedTo.Valid && t.AssignedTo.UUID != claims.UserID {
		assignee := t.AssignedTo.UUID
		s.notifSvc.NotifyAsync(context.Background(), notifications.NotifyInput{
			RecipientID: assignee,
			Type:        db.NotificationTypeTaskAssigned,
			Priority:    db.NotificationPriorityMedium,
			Title:       "Task assigned to you",
			Message:     t.Title,
			LinkURL:     "/cases/" + caseID.String() + "/tasks/" + t.ID.String(),
			CaseID:      &caseID,
			TaskID:      &t.ID,
		})
	}

	return s.Get(ctx, t.ID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*TaskResponse, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	count, _ := s.repo.CountComments(ctx, id)
	return s.toResponseFromRow(row, count), nil
}

func (s *Service) ListByCase(ctx context.Context, caseID uuid.UUID, f ListFilters) ([]TaskResponse, ListMeta, error) {
	rows, err := s.repo.ListByCase(ctx, caseID, f)
	if err != nil {
		return nil, ListMeta{}, err
	}

	limit := int(f.Limit)
	if limit <= 0 {
		limit = 50
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.CountByCase(ctx, caseID, f.Status)

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c := encodeCursor(last.SortOrder, last.CreatedAt, last.ID)
		nextCursor = &c
	}

	out := make([]TaskResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponseFromListRow(r))
	}
	return out, ListMeta{TotalCount: total, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *Service) ListMyTasks(ctx context.Context, userID uuid.UUID, f ListFilters) ([]TaskResponse, ListMeta, error) {
	rows, err := s.repo.ListMyTasks(ctx, userID, f)
	if err != nil {
		return nil, ListMeta{}, err
	}

	limit := int(f.Limit)
	if limit <= 0 {
		limit = 50
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	out := make([]TaskResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, s.toResponseFromMyTaskRow(r))
	}
	return out, ListMeta{TotalCount: int64(len(out)), HasMore: hasMore}, nil
}

const listAllTasksSQL = `
SELECT
    t.id, t.case_id, t.created_by, t.assigned_to, t.timeline_event_id,
    t.title, t.description, t.status, t.priority, t.due_date, t.completed_at,
    t.is_overdue, t.is_ai_suggested, t.sort_order,
    t.created_at, t.updated_at, t.deleted_at, t.search_vector,
    c.case_number, c.title AS case_title,
    u_creator.full_name AS created_by_name,
    u_assign.full_name  AS assigned_to_name,
    u_assign.email      AS assigned_to_email
FROM tasks t
JOIN cases c ON c.id = t.case_id AND c.deleted_at IS NULL
JOIN users u_creator ON u_creator.id = t.created_by
LEFT JOIN users u_assign ON u_assign.id = t.assigned_to
WHERE t.deleted_at IS NULL
  AND ($1::text IS NULL OR t.status::text = $1)
  AND ($2 IS NOT TRUE OR t.is_overdue = TRUE)
  AND ($3::uuid IS NULL OR t.assigned_to = $3)
  AND ($4 IS NOT TRUE OR t.assigned_to IS NULL)
ORDER BY t.is_overdue DESC, t.due_date ASC NULLS LAST, t.priority ASC
LIMIT $5
`

func (s *Service) ListAllTasks(ctx context.Context, f ListFilters) ([]TaskResponse, ListMeta, error) {
	if s.sqlDB == nil {
		return nil, ListMeta{}, errors.New("database not available for all-tasks view")
	}

	limit := int32(50)
	if f.Limit > 0 && f.Limit <= 200 {
		limit = f.Limit
	}

	var statusVal interface{}
	if f.Status != nil {
		statusVal = string(*f.Status)
	}
	var assignedToVal interface{}
	if f.AssignedTo != nil {
		assignedToVal = *f.AssignedTo
	}

	rows, err := s.sqlDB.QueryContext(ctx, listAllTasksSQL, statusVal, f.OverdueOnly, assignedToVal, f.UnassignedOnly, limit+1)
	if err != nil {
		return nil, ListMeta{}, err
	}
	defer rows.Close()

	var tasks []TaskResponse
	for rows.Next() {
		var (
			id, caseID, createdBy uuid.UUID
			assignedTo            uuid.NullUUID
			timelineEventID       uuid.NullUUID
			title                 string
			description           sql.NullString
			status                db.TaskStatus
			priority              db.TaskPriority
			dueDate, completedAt  sql.NullTime
			isOverdue, isAI       bool
			sortOrder             int32
			createdAt, updatedAt  time.Time
			deletedAt             sql.NullTime
			searchVector          interface{}
			caseNumber, caseTitle string
			createdByName         string
			assignedToName        sql.NullString
			assignedToEmail       sql.NullString
		)
		if err := rows.Scan(
			&id, &caseID, &createdBy, &assignedTo, &timelineEventID,
			&title, &description, &status, &priority, &dueDate, &completedAt,
			&isOverdue, &isAI, &sortOrder,
			&createdAt, &updatedAt, &deletedAt, &searchVector,
			&caseNumber, &caseTitle,
			&createdByName, &assignedToName, &assignedToEmail,
		); err != nil {
			return nil, ListMeta{}, err
		}
		r := TaskResponse{
			ID:            id,
			CaseID:        caseID,
			CaseNumber:    caseNumber,
			CaseTitle:     caseTitle,
			Title:         title,
			Description:   description.String,
			Status:        status,
			Priority:      priority,
			IsOverdue:     isOverdue,
			IsAISuggested: isAI,
			SortOrder:     sortOrder,
			DueDate:       timePtr(dueDate),
			CompletedAt:   timePtr(completedAt),
			CreatedBy:     UserRef{ID: createdBy, FullName: createdByName},
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		}
		if assignedTo.Valid {
			ref := &UserRef{ID: assignedTo.UUID}
			if assignedToName.Valid {
				ref.FullName = assignedToName.String
			}
			if assignedToEmail.Valid {
				ref.Email = assignedToEmail.String
			}
			r.AssignedTo = ref
		}
		tasks = append(tasks, r)
	}
	if err := rows.Err(); err != nil {
		return nil, ListMeta{}, err
	}

	hasMore := len(tasks) > int(limit)
	if hasMore {
		tasks = tasks[:limit]
	}
	return tasks, ListMeta{TotalCount: int64(len(tasks)), HasMore: hasMore}, nil
}

func (s *Service) Update(ctx context.Context, claims *auth.Claims, id uuid.UUID, req UpdateTaskRequest) (*TaskResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Only assignee, creator, or manager+ can update
	actorRole := claims.Role
	if actorRole != db.UserRoleManager && actorRole != db.UserRoleAdmin && actorRole != db.UserRoleSuperAdmin {
		if existing.CreatedBy != claims.UserID &&
			!(existing.AssignedTo.Valid && existing.AssignedTo.UUID == claims.UserID) {
			return nil, ErrForbidden
		}
	}

	// Validate status transition if changing status
	if req.Status != nil {
		if err := validateTransition(existing.Status, *req.Status); err != nil {
			return nil, err
		}
		// Block in_progress if a prerequisite playbook step is not yet done
		if *req.Status == db.TaskStatusInProgress && s.isBlocked(ctx, id) {
			return nil, ErrBlocked
		}
	}

	p := db.UpdateTaskParams{ID: id}
	if req.Title != nil {
		p.Title = sql.NullString{String: *req.Title, Valid: true}
	}
	if req.Description != nil {
		p.Description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.Priority != nil {
		p.Priority = db.NullTaskPriority{TaskPriority: *req.Priority, Valid: true}
	}
	if req.ClearDueDate {
		p.ClearDueDate = sql.NullBool{Bool: true, Valid: true}
	} else if req.DueDate != nil {
		p.DueDate = sql.NullTime{Time: *req.DueDate, Valid: true}
		// Update overdue flag
		p.IsOverdue = sql.NullBool{Bool: req.DueDate.Before(time.Now()), Valid: true}
	}
	if req.ClearAssignedTo {
		p.ClearAssignedTo = sql.NullBool{Bool: true, Valid: true}
	} else if req.AssignedTo != nil {
		p.AssignedTo = uuid.NullUUID{UUID: *req.AssignedTo, Valid: true}
	}
	if req.Status != nil {
		p.Status = db.NullTaskStatus{TaskStatus: *req.Status, Valid: true}
		if *req.Status == db.TaskStatusDone {
			now := time.Now()
			p.CompletedAt = sql.NullTime{Time: now, Valid: true}
			p.IsOverdue = sql.NullBool{Bool: false, Valid: true}
		}
	}
	if req.SortOrder != nil {
		p.SortOrder = sql.NullInt32{Int32: *req.SortOrder, Valid: true}
	}

	if _, err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	// Emit timeline event when status changes (e.g. undo-complete, reopen, in_progress)
	if req.Status != nil && *req.Status != existing.Status {
		actorID := claims.UserID
		go s.emitter.Emit(context.Background(), timeline.Event{
			CaseID:    existing.CaseID,
			EventType: db.TimelineEventTypeTaskStatusChanged,
			Source:    db.TimelineEventSourceUser,
			ActorID:   &actorID,
			OccurredAt: time.Now(),
			Payload: map[string]any{
				"task_id":    id.String(),
				"title":      existing.Title,
				"from_status": string(existing.Status),
				"to_status":  string(*req.Status),
			},
		})
	}

	// Notify new assignee if the assignment changed
	if s.notifSvc != nil && req.AssignedTo != nil {
		newAssignee := *req.AssignedTo
		prevAssignee := existing.AssignedTo
		if newAssignee != claims.UserID && (!prevAssignee.Valid || prevAssignee.UUID != newAssignee) {
			caseID := existing.CaseID
			go s.notifSvc.NotifyAsync(context.Background(), notifications.NotifyInput{
				RecipientID: newAssignee,
				Type:        db.NotificationTypeTaskAssigned,
				Priority:    db.NotificationPriorityMedium,
				Title:       "Task assigned to you",
				Message:     existing.Title,
				LinkURL:     "/cases/" + caseID.String() + "/tasks/" + id.String(),
				CaseID:      &caseID,
				TaskID:      &id,
			})
		}
	}

	return s.Get(ctx, id)
}

func (s *Service) Complete(ctx context.Context, claims *auth.Claims, id uuid.UUID) (*TaskResponse, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	if existing.Status == db.TaskStatusDone {
		// idempotent
		count, _ := s.repo.CountComments(ctx, id)
		return s.toResponseFromRow(existing, count), nil
	}
	if existing.Status == db.TaskStatusCancelled {
		return nil, ErrInvalidTransition
	}

	// If the task is assigned, only the assignee or manager+ may complete it.
	// This ensures the person responsible for the work is the one confirming it is done.
	actorRole := claims.Role
	if existing.AssignedTo.Valid && existing.AssignedTo.UUID != claims.UserID {
		if actorRole != db.UserRoleManager && actorRole != db.UserRoleAdmin && actorRole != db.UserRoleSuperAdmin {
			return nil, ErrForbidden
		}
	}

	if _, err := s.repo.Complete(ctx, id); err != nil {
		return nil, err
	}

	actorID := claims.UserID
	go s.emitter.Emit(context.Background(), timeline.Event{
		CaseID:    existing.CaseID,
		EventType: db.TimelineEventTypeTaskCompleted,
		Source:    db.TimelineEventSourceUser,
		ActorID:   &actorID,
		OccurredAt: time.Now(),
		Payload: map[string]any{
			"task_id": id.String(),
			"title":   existing.Title,
		},
	})

	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, claims *auth.Claims, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	actorRole := claims.Role
	if actorRole != db.UserRoleManager && actorRole != db.UserRoleAdmin && actorRole != db.UserRoleSuperAdmin {
		if existing.CreatedBy != claims.UserID {
			return ErrForbidden
		}
	}
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) AddComment(ctx context.Context, claims *auth.Claims, taskID uuid.UUID, req CreateCommentRequest) (*CommentResponse, error) {
	if req.Content == "" {
		return nil, ErrTitleRequired
	}
	c, err := s.repo.CreateComment(ctx, db.CreateTaskCommentParams{
		TaskID:   taskID,
		AuthorID: claims.UserID,
		Content:  req.Content,
	})
	if err != nil {
		return nil, err
	}
	return &CommentResponse{
		ID:        c.ID,
		TaskID:    c.TaskID,
		Author:    UserRef{ID: claims.UserID, Email: claims.Email},
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func (s *Service) ListComments(ctx context.Context, taskID uuid.UUID, cursorID *uuid.UUID, limit int32) ([]CommentResponse, bool, *uuid.UUID, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.repo.ListComments(ctx, taskID, cursorID, limit)
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

	out := make([]CommentResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, CommentResponse{
			ID:     r.ID,
			TaskID: r.TaskID,
			Author: UserRef{
				ID:       r.AuthorID,
				FullName: r.AuthorName,
				Email:    r.AuthorEmail,
			},
			Content:   r.Content,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	return out, hasMore, nextCursor, nil
}

// ── Playbook task creation ────────────────────────────────────────────────────

const createPlaybookTaskSQL = `
INSERT INTO tasks (case_id, created_by, title, description, priority, due_date, playbook_step_id, playbook_step_order, depends_on_step)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

// CreateRaw implements playbooks.TaskCreator, creating tasks with playbook metadata.
func (s *Service) CreateRaw(ctx context.Context, p playbooks.CreateTaskRawParams) error {
	if s.sqlDB == nil {
		return errors.New("database not available for raw task creation")
	}
	var dueDate sql.NullTime
	if p.DueDate != nil {
		dueDate = sql.NullTime{Time: *p.DueDate, Valid: true}
	}
	var stepID interface{}
	if p.PlaybookStepID != nil {
		stepID = *p.PlaybookStepID
	}
	_, err := s.sqlDB.ExecContext(ctx, createPlaybookTaskSQL,
		p.CaseID, p.CreatedBy, p.Title, p.Description, p.Priority,
		dueDate, stepID, p.PlaybookStepOrder, p.DependsOnStep,
	)
	return err
}

const getTaskDependsOnSQL = `SELECT depends_on_step, case_id FROM tasks WHERE id = $1 AND deleted_at IS NULL`

const checkBlockedSQL = `
SELECT EXISTS(
  SELECT 1 FROM tasks
  WHERE case_id = $1 AND playbook_step_order = $2 AND status != 'done' AND deleted_at IS NULL
)`

// isBlocked returns true if a task with depends_on_step > 0 has an incomplete predecessor.
func (s *Service) isBlocked(ctx context.Context, taskID uuid.UUID) bool {
	if s.sqlDB == nil {
		return false
	}
	var dependsOn int32
	var caseID uuid.UUID
	if err := s.sqlDB.QueryRowContext(ctx, getTaskDependsOnSQL, taskID).Scan(&dependsOn, &caseID); err != nil || dependsOn == 0 {
		return false
	}
	var blocked bool
	s.sqlDB.QueryRowContext(ctx, checkBlockedSQL, caseID, dependsOn).Scan(&blocked) //nolint:errcheck
	return blocked
}

// ── Response builders ─────────────────────────────────────────────────────────

func (s *Service) toResponse(t db.Task, creatorName, creatorEmail string, commentCount *int64) *TaskResponse {
	r := &TaskResponse{
		ID:            t.ID,
		CaseID:        t.CaseID,
		Title:         t.Title,
		Description:   t.Description.String,
		Status:        t.Status,
		Priority:      t.Priority,
		IsOverdue:     t.IsOverdue,
		IsAISuggested: t.IsAiSuggested,
		SortOrder:     t.SortOrder,
		DueDate:       timePtr(t.DueDate),
		CompletedAt:   timePtr(t.CompletedAt),
		CreatedBy:     UserRef{ID: t.CreatedBy, FullName: creatorName, Email: creatorEmail},
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}
	if t.AssignedTo.Valid {
		r.AssignedTo = &UserRef{ID: t.AssignedTo.UUID}
	}
	if commentCount != nil {
		r.CommentCount = *commentCount
	}
	return r
}

func (s *Service) toResponseFromRow(row db.GetTaskByIDRow, commentCount int64) *TaskResponse {
	var assignee *UserRef
	if row.AssignedTo.Valid {
		ref := &UserRef{ID: row.AssignedTo.UUID}
		if row.AssignedToName.Valid {
			ref.FullName = row.AssignedToName.String
		}
		if row.AssignedToEmail.Valid {
			ref.Email = row.AssignedToEmail.String
		}
		if row.AssignedToRole.Valid {
			ref.Role = string(row.AssignedToRole.UserRole)
		}
		assignee = ref
	}
	return &TaskResponse{
		ID:            row.ID,
		CaseID:        row.CaseID,
		Title:         row.Title,
		Description:   row.Description.String,
		Status:        row.Status,
		Priority:      row.Priority,
		IsOverdue:     row.IsOverdue,
		IsAISuggested: row.IsAiSuggested,
		SortOrder:     row.SortOrder,
		DueDate:       timePtr(row.DueDate),
		CompletedAt:   timePtr(row.CompletedAt),
		CreatedBy:     UserRef{ID: row.CreatedBy, FullName: row.CreatedByName, Email: row.CreatedByEmail},
		AssignedTo:    assignee,
		CommentCount:  commentCount,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func (s *Service) toResponseFromListRow(row db.ListTasksByCaseRow) TaskResponse {
	r := TaskResponse{
		ID:            row.ID,
		CaseID:        row.CaseID,
		Title:         row.Title,
		Description:   row.Description.String,
		Status:        row.Status,
		Priority:      row.Priority,
		IsOverdue:     row.IsOverdue,
		IsAISuggested: row.IsAiSuggested,
		SortOrder:     row.SortOrder,
		DueDate:       timePtr(row.DueDate),
		CompletedAt:   timePtr(row.CompletedAt),
		CreatedBy:     UserRef{ID: row.CreatedBy, FullName: row.CreatedByName},
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.AssignedTo.Valid {
		ref := &UserRef{ID: row.AssignedTo.UUID}
		if row.AssignedToName.Valid {
			ref.FullName = row.AssignedToName.String
		}
		if row.AssignedToEmail.Valid {
			ref.Email = row.AssignedToEmail.String
		}
		r.AssignedTo = ref
	}
	return r
}

func (s *Service) toResponseFromMyTaskRow(row db.ListMyTasksRow) TaskResponse {
	r := TaskResponse{
		ID:            row.ID,
		CaseID:        row.CaseID,
		CaseNumber:    row.CaseNumber,
		CaseTitle:     row.CaseTitle,
		Title:         row.Title,
		Description:   row.Description.String,
		Status:        row.Status,
		Priority:      row.Priority,
		IsOverdue:     row.IsOverdue,
		IsAISuggested: row.IsAiSuggested,
		SortOrder:     row.SortOrder,
		DueDate:       timePtr(row.DueDate),
		CompletedAt:   timePtr(row.CompletedAt),
		CreatedBy:     UserRef{ID: row.CreatedBy, FullName: row.CreatedByName},
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.AssignedTo.Valid {
		ref := &UserRef{ID: row.AssignedTo.UUID}
		if row.AssignedToName.Valid {
			ref.FullName = row.AssignedToName.String
		}
		if row.AssignedToEmail.Valid {
			ref.Email = row.AssignedToEmail.String
		}
		r.AssignedTo = ref
	}
	return r
}
