package cases

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q     *db.Queries
	sqlDB *sql.DB
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) WithDB(sqlDB *sql.DB) *Repository {
	r.sqlDB = sqlDB
	return r
}

func (r *Repository) NextCaseSeq(ctx context.Context) (int64, error) {
	return r.q.NextCaseSeq(ctx)
}

const listActiveCasesSQL = `
SELECT
  c.id, c.case_number, c.title, c.description, c.status, c.priority, c.assigned_to, c.created_by, c.organization_id, c.due_date, c.sla_breach_at, c.sla_breached, c.tags, c.created_at, c.updated_at, c.deleted_at, c.search_vector,
  creator.full_name  AS created_by_name,
  assignee.full_name AS assigned_to_name,
  ci.risk_score, ci.risk_level
FROM cases c
JOIN users creator ON creator.id = c.created_by
LEFT JOIN users assignee ON assignee.id = c.assigned_to
LEFT JOIN LATERAL (
  SELECT risk_score, risk_level FROM case_ai_insights
  WHERE case_id = c.id ORDER BY analysed_at DESC LIMIT 1
) ci ON true
WHERE c.deleted_at IS NULL
  AND c.organization_id = $1
  AND c.status NOT IN ('draft', 'closed')
  AND ($2::case_priority IS NULL OR c.priority = $2::case_priority)
  AND ($3::uuid IS NULL OR c.assigned_to = $3)
  AND ($4::boolean IS NULL OR c.sla_breached = $4)
  AND ($5::uuid IS NULL OR c.id < $5)
  AND (
    $6::uuid IS NULL
    OR c.assigned_to = $6
    OR c.created_by  = $6
    OR EXISTS (SELECT 1 FROM case_watchers cw WHERE cw.case_id = c.id AND cw.user_id = $6)
  )
ORDER BY c.created_at DESC
LIMIT $7
`

const countActiveCasesSQL = `
SELECT COUNT(*) FROM cases c
WHERE c.deleted_at IS NULL
  AND c.organization_id = $1
  AND c.status NOT IN ('draft', 'closed')
  AND ($2::case_priority IS NULL OR c.priority = $2::case_priority)
  AND ($3::uuid IS NULL OR c.assigned_to = $3)
  AND ($4::boolean IS NULL OR c.sla_breached = $4)
  AND (
    $5::uuid IS NULL
    OR c.assigned_to = $5
    OR c.created_by  = $5
    OR EXISTS (SELECT 1 FROM case_watchers cw WHERE cw.case_id = c.id AND cw.user_id = $5)
  )
`

func (r *Repository) ListActive(ctx context.Context, f ListFilters, orgID uuid.UUID) ([]db.ListCasesRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var priority interface{}
	if f.Priority != nil {
		priority = string(*f.Priority)
	}
	var assignedTo interface{}
	if f.AssignedTo != nil {
		assignedTo = *f.AssignedTo
	}
	var slaBreached interface{}
	if f.SLABreached != nil {
		slaBreached = *f.SLABreached
	}
	var cursor interface{}
	if f.Cursor != nil {
		cursor = *f.Cursor
	}
	var viewerID interface{}
	if f.ViewerID != nil {
		viewerID = *f.ViewerID
	}

	rows, err := r.sqlDB.QueryContext(ctx, listActiveCasesSQL,
		orgID, priority, assignedTo, slaBreached, cursor, viewerID, limit+1,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []db.ListCasesRow
	for rows.Next() {
		var i db.ListCasesRow
		if err := rows.Scan(
			&i.ID, &i.CaseNumber, &i.Title, &i.Description, &i.Status, &i.Priority,
			&i.AssignedTo, &i.CreatedBy, &i.OrganizationID, &i.DueDate, &i.SlaBreachAt,
			&i.SlaBreached, pq.Array(&i.Tags), &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
			&i.SearchVector, &i.CreatedByName, &i.AssignedToName, &i.RiskScore, &i.RiskLevel,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *Repository) CountActive(ctx context.Context, f ListFilters, orgID uuid.UUID) (int64, error) {
	var priority interface{}
	if f.Priority != nil {
		priority = string(*f.Priority)
	}
	var assignedTo interface{}
	if f.AssignedTo != nil {
		assignedTo = *f.AssignedTo
	}
	var slaBreached interface{}
	if f.SLABreached != nil {
		slaBreached = *f.SLABreached
	}
	var viewerID interface{}
	if f.ViewerID != nil {
		viewerID = *f.ViewerID
	}

	var count int64
	err := r.sqlDB.QueryRowContext(ctx, countActiveCasesSQL,
		orgID, priority, assignedTo, slaBreached, viewerID,
	).Scan(&count)
	return count, err
}

const listByStatusesSQL = `
SELECT
  c.id, c.case_number, c.title, c.description, c.status, c.priority, c.assigned_to, c.created_by, c.organization_id, c.due_date, c.sla_breach_at, c.sla_breached, c.tags, c.created_at, c.updated_at, c.deleted_at, c.search_vector,
  creator.full_name  AS created_by_name,
  assignee.full_name AS assigned_to_name,
  ci.risk_score, ci.risk_level
FROM cases c
JOIN users creator ON creator.id = c.created_by
LEFT JOIN users assignee ON assignee.id = c.assigned_to
LEFT JOIN LATERAL (
  SELECT risk_score, risk_level FROM case_ai_insights
  WHERE case_id = c.id ORDER BY analysed_at DESC LIMIT 1
) ci ON true
WHERE c.deleted_at IS NULL
  AND c.organization_id = $1
  AND c.status = ANY($2::case_status[])
  AND ($3::case_priority IS NULL OR c.priority = $3::case_priority)
  AND ($4::uuid IS NULL OR c.assigned_to = $4)
  AND ($5::boolean IS NULL OR c.sla_breached = $5)
  AND ($6::uuid IS NULL OR c.id < $6)
  AND (
    $7::uuid IS NULL
    OR c.assigned_to = $7
    OR c.created_by  = $7
    OR EXISTS (SELECT 1 FROM case_watchers cw WHERE cw.case_id = c.id AND cw.user_id = $7)
  )
ORDER BY c.created_at DESC
LIMIT $8
`

const countByStatusesSQL = `
SELECT COUNT(*) FROM cases c
WHERE c.deleted_at IS NULL
  AND c.organization_id = $1
  AND c.status = ANY($2::case_status[])
  AND ($3::case_priority IS NULL OR c.priority = $3::case_priority)
  AND ($4::uuid IS NULL OR c.assigned_to = $4)
  AND ($5::boolean IS NULL OR c.sla_breached = $5)
  AND (
    $6::uuid IS NULL
    OR c.assigned_to = $6
    OR c.created_by  = $6
    OR EXISTS (SELECT 1 FROM case_watchers cw WHERE cw.case_id = c.id AND cw.user_id = $6)
  )
`

func (r *Repository) ListByStatuses(ctx context.Context, f ListFilters, orgID uuid.UUID) ([]db.ListCasesRow, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	statuses := make([]string, len(f.Statuses))
	for i, s := range f.Statuses {
		statuses[i] = string(s)
	}
	var priority interface{}
	if f.Priority != nil {
		priority = string(*f.Priority)
	}
	var assignedTo interface{}
	if f.AssignedTo != nil {
		assignedTo = *f.AssignedTo
	}
	var slaBreached interface{}
	if f.SLABreached != nil {
		slaBreached = *f.SLABreached
	}
	var cursor interface{}
	if f.Cursor != nil {
		cursor = *f.Cursor
	}
	var viewerID interface{}
	if f.ViewerID != nil {
		viewerID = *f.ViewerID
	}

	rows, err := r.sqlDB.QueryContext(ctx, listByStatusesSQL,
		orgID, pq.Array(statuses), priority, assignedTo, slaBreached, cursor, viewerID, limit+1,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []db.ListCasesRow
	for rows.Next() {
		var i db.ListCasesRow
		if err := rows.Scan(
			&i.ID, &i.CaseNumber, &i.Title, &i.Description, &i.Status, &i.Priority,
			&i.AssignedTo, &i.CreatedBy, &i.OrganizationID, &i.DueDate, &i.SlaBreachAt,
			&i.SlaBreached, pq.Array(&i.Tags), &i.CreatedAt, &i.UpdatedAt, &i.DeletedAt,
			&i.SearchVector, &i.CreatedByName, &i.AssignedToName, &i.RiskScore, &i.RiskLevel,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (r *Repository) CountByStatuses(ctx context.Context, f ListFilters, orgID uuid.UUID) (int64, error) {
	statuses := make([]string, len(f.Statuses))
	for i, s := range f.Statuses {
		statuses[i] = string(s)
	}
	var priority interface{}
	if f.Priority != nil {
		priority = string(*f.Priority)
	}
	var assignedTo interface{}
	if f.AssignedTo != nil {
		assignedTo = *f.AssignedTo
	}
	var slaBreached interface{}
	if f.SLABreached != nil {
		slaBreached = *f.SLABreached
	}
	var viewerID interface{}
	if f.ViewerID != nil {
		viewerID = *f.ViewerID
	}

	var count int64
	err := r.sqlDB.QueryRowContext(ctx, countByStatusesSQL,
		orgID, pq.Array(statuses), priority, assignedTo, slaBreached, viewerID,
	).Scan(&count)
	return count, err
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
	if f.ViewerID != nil {
		p.ViewerID = uuid.NullUUID{UUID: *f.ViewerID, Valid: true}
	}
	if f.Q != nil && *f.Q != "" {
		p.Q = sql.NullString{String: *f.Q, Valid: true}
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
	if f.ViewerID != nil {
		p.ViewerID = uuid.NullUUID{UUID: *f.ViewerID, Valid: true}
	}
	return r.q.CountCases(ctx, p)
}

func (r *Repository) GetLastPendingReviewActor(ctx context.Context, caseID uuid.UUID) (uuid.UUID, error) {
	return r.q.GetLastPendingReviewActor(ctx, caseID)
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

// ListManagers returns all active users with manager, admin, or super_admin roles.
// Used to fan-out pending_review notifications to supervisors.
func (r *Repository) ListManagers(ctx context.Context) ([]db.ListUsersRow, error) {
	var result []db.ListUsersRow
	for _, role := range []db.UserRole{db.UserRoleManager, db.UserRoleAdmin, db.UserRoleSuperAdmin} {
		rows, err := r.q.ListUsers(ctx, db.ListUsersParams{
			Limit: 200,
			Role:  db.NullUserRole{UserRole: role, Valid: true},
		})
		if err != nil {
			return nil, err
		}
		for _, u := range rows {
			if u.IsActive && u.DeletedAt.Time.IsZero() {
				result = append(result, u)
			}
		}
	}
	return result, nil
}
