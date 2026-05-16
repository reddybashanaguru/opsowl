package cases

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) Create(ctx context.Context, p db.CreateCaseParams) (db.Case, error) {
	return r.q.CreateCase(ctx, p)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.GetCaseByIDRow, error) {
	return r.q.GetCaseByID(ctx, id)
}

func (r *Repository) List(ctx context.Context, f ListFilters, orgID uuid.UUID) ([]db.ListCasesRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	p := db.ListCasesParams{
		OrganizationID: orgID,
		RowLimit:       limit + 1,
	}
	if f.Status != nil {
		p.Status = db.NullCaseStatus{CaseStatus: *f.Status, Valid: true}
	}
	if f.Priority != nil {
		p.Priority = db.NullCasePriority{CasePriority: *f.Priority, Valid: true}
	}
	if f.AssignedTo != nil {
		p.AssignedTo = uuid.NullUUID{UUID: *f.AssignedTo, Valid: true}
	}
	if f.SLABreached != nil {
		p.SlaBreached = sql.NullBool{Bool: *f.SLABreached, Valid: true}
	}
	if f.From != nil {
		p.FromTime = sql.NullTime{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		p.ToTime = sql.NullTime{Time: *f.To, Valid: true}
	}
	if f.Cursor != nil {
		p.Cursor = uuid.NullUUID{UUID: *f.Cursor, Valid: true}
	}
	return r.q.ListCases(ctx, p)
}

func (r *Repository) Count(ctx context.Context, f ListFilters, orgID uuid.UUID) (int64, error) {
	p := db.CountCasesParams{OrganizationID: orgID}
	if f.Status != nil {
		p.Status = db.NullCaseStatus{CaseStatus: *f.Status, Valid: true}
	}
	if f.Priority != nil {
		p.Priority = db.NullCasePriority{CasePriority: *f.Priority, Valid: true}
	}
	if f.AssignedTo != nil {
		p.AssignedTo = uuid.NullUUID{UUID: *f.AssignedTo, Valid: true}
	}
	if f.SLABreached != nil {
		p.SlaBreached = sql.NullBool{Bool: *f.SLABreached, Valid: true}
	}
	return r.q.CountCases(ctx, p)
}

func (r *Repository) Update(ctx context.Context, p db.UpdateCaseParams) (db.Case, error) {
	return r.q.UpdateCase(ctx, p)
}

func (r *Repository) UpdateStatus(ctx context.Context, p db.UpdateCaseStatusParams) (db.Case, error) {
	return r.q.UpdateCaseStatus(ctx, p)
}

func (r *Repository) UpdateAssignment(ctx context.Context, p db.UpdateCaseAssignmentParams) (db.Case, error) {
	return r.q.UpdateCaseAssignment(ctx, p)
}

func (r *Repository) UpdateTags(ctx context.Context, p db.UpdateCaseTagsParams) (db.Case, error) {
	return r.q.UpdateCaseTags(ctx, p)
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) (db.Case, error) {
	return r.q.SoftDeleteCase(ctx, id)
}

func (r *Repository) CreateActivity(ctx context.Context, caseID, actorID uuid.UUID, eventType string, payload map[string]interface{}) (db.CaseActivity, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return db.CaseActivity{}, err
	}
	return r.q.CreateCaseActivity(ctx, db.CreateCaseActivityParams{
		CaseID:    caseID,
		ActorID:   actorID,
		EventType: eventType,
		Payload:   raw,
	})
}

func (r *Repository) ListActivities(ctx context.Context, caseID uuid.UUID, cursor *uuid.UUID, limit int32) ([]db.ListCaseActivitiesRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	p := db.ListCaseActivitiesParams{
		CaseID:   caseID,
		RowLimit: limit + 1,
	}
	if cursor != nil {
		p.Cursor = uuid.NullUUID{UUID: *cursor, Valid: true}
	}
	return r.q.ListCaseActivities(ctx, p)
}

func (r *Repository) CreateAssignment(ctx context.Context, caseID, assignedTo, assignedBy uuid.UUID) error {
	if err := r.q.CloseLastAssignment(ctx, caseID); err != nil {
		return err
	}
	_, err := r.q.CreateCaseAssignment(ctx, db.CreateCaseAssignmentParams{
		CaseID:     caseID,
		AssignedTo: assignedTo,
		AssignedBy: assignedBy,
	})
	return err
}

func (r *Repository) UpsertTag(ctx context.Context, name string) (db.CaseTag, error) {
	return r.q.UpsertCaseTag(ctx, name)
}

func (r *Repository) AddWatcher(ctx context.Context, caseID, userID uuid.UUID) error {
	return r.q.AddCaseWatcher(ctx, db.AddCaseWatcherParams{CaseID: caseID, UserID: userID})
}

func (r *Repository) GetWatchers(ctx context.Context, caseID uuid.UUID) ([]db.GetCaseWatchersRow, error) {
	return r.q.GetCaseWatchers(ctx, caseID)
}

func nullTimeFromPtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
