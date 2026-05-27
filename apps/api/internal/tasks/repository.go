package tasks

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) Create(ctx context.Context, p db.CreateTaskParams) (db.Task, error) {
	return r.q.CreateTask(ctx, p)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.GetTaskByIDRow, error) {
	return r.q.GetTaskByID(ctx, id)
}

func (r *Repository) ListByCase(ctx context.Context, caseID uuid.UUID, f ListFilters) ([]db.ListTasksByCaseRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	p := db.ListTasksByCaseParams{
		CaseID:   caseID,
		RowLimit: limit + 1,
	}
	if f.Status != nil {
		p.Status = db.NullTaskStatus{TaskStatus: *f.Status, Valid: true}
	}
	if f.Priority != nil {
		p.Priority = db.NullTaskPriority{TaskPriority: *f.Priority, Valid: true}
	}
	if f.AssignedTo != nil {
		p.AssignedTo = uuid.NullUUID{UUID: *f.AssignedTo, Valid: true}
	}
	if f.OverdueOnly {
		p.OverdueOnly = sql.NullBool{Bool: true, Valid: true}
	}
	if f.CursorSortOrder != nil && f.CursorTime != nil && f.CursorID != nil {
		p.CursorSortOrder = sql.NullInt32{Int32: *f.CursorSortOrder, Valid: true}
		p.CursorTime = sql.NullTime{Time: *f.CursorTime, Valid: true}
		p.CursorID = uuid.NullUUID{UUID: *f.CursorID, Valid: true}
	}
	return r.q.ListTasksByCase(ctx, p)
}

func (r *Repository) ListMyTasks(ctx context.Context, userID uuid.UUID, f ListFilters) ([]db.ListMyTasksRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	p := db.ListMyTasksParams{
		AssignedTo: uuid.NullUUID{UUID: userID, Valid: true},
		RowLimit:   limit + 1,
	}
	if f.Status != nil {
		p.Status = db.NullTaskStatus{TaskStatus: *f.Status, Valid: true}
	}
	if f.OverdueOnly {
		p.OverdueOnly = sql.NullBool{Bool: true, Valid: true}
	}
	return r.q.ListMyTasks(ctx, p)
}

func (r *Repository) CountByCase(ctx context.Context, caseID uuid.UUID, status *db.TaskStatus) (int64, error) {
	p := db.CountTasksByCaseParams{CaseID: caseID}
	if status != nil {
		p.Status = db.NullTaskStatus{TaskStatus: *status, Valid: true}
	}
	return r.q.CountTasksByCase(ctx, p)
}

func (r *Repository) Update(ctx context.Context, p db.UpdateTaskParams) (db.Task, error) {
	return r.q.UpdateTask(ctx, p)
}

func (r *Repository) Complete(ctx context.Context, id uuid.UUID) (db.Task, error) {
	return r.q.CompleteTask(ctx, id)
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.SoftDeleteTask(ctx, id)
	return err
}

func (r *Repository) CountComments(ctx context.Context, taskID uuid.UUID) (int64, error) {
	return r.q.CountTaskComments(ctx, taskID)
}

func (r *Repository) CreateComment(ctx context.Context, p db.CreateTaskCommentParams) (db.TaskComment, error) {
	return r.q.CreateTaskComment(ctx, p)
}

func (r *Repository) ListComments(ctx context.Context, taskID uuid.UUID, cursorID *uuid.UUID, limit int32) ([]db.ListTaskCommentsRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	p := db.ListTaskCommentsParams{
		TaskID:   taskID,
		RowLimit: limit + 1,
	}
	if cursorID != nil {
		p.CursorID = uuid.NullUUID{UUID: *cursorID, Valid: true}
	}
	return r.q.ListTaskComments(ctx, p)
}
