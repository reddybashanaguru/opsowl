package playbooks

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// playbookRow matches the columns returned by the main playbooks queries.
type playbookRow struct {
	ID          uuid.UUID
	Name        string
	Description string
	CaseType    string
	IsActive    bool
	OrgID       uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// stepRow matches the columns returned by the step queries.
type stepRow struct {
	ID            uuid.UUID
	PlaybookID    uuid.UUID
	Title         string
	Description   string
	Priority      db.TaskPriority
	StepOrder     int
	DueHours      sql.NullInt32
	DependsOnStep int
	CreatedAt     time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(sqlDB *sql.DB) *Repository {
	return &Repository{db: sqlDB}
}

const insertPlaybookSQL = `
INSERT INTO playbooks (name, description, case_type, org_id, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, description, case_type, is_active, org_id, created_at, updated_at
`

func (r *Repository) Create(ctx context.Context, name, description, caseType string, orgID, createdBy uuid.UUID) (playbookRow, error) {
	row := r.db.QueryRowContext(ctx, insertPlaybookSQL, name, description, caseType, orgID, createdBy)
	return scanPlaybook(row)
}

const insertStepSQL = `
INSERT INTO playbook_steps (playbook_id, title, description, priority, step_order, due_hours, depends_on_step)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, playbook_id, title, description, priority, step_order, due_hours, depends_on_step, created_at
`

func (r *Repository) CreateStep(ctx context.Context, playbookID uuid.UUID, s CreateStepRequest) (stepRow, error) {
	var dueHours sql.NullInt32
	if s.DueHours != nil {
		dueHours = sql.NullInt32{Int32: int32(*s.DueHours), Valid: true}
	}
	priority := s.Priority
	if priority == "" {
		priority = db.TaskPriorityMedium
	}
	row := r.db.QueryRowContext(ctx, insertStepSQL,
		playbookID, s.Title, s.Description, priority,
		s.StepOrder, dueHours, s.DependsOnStep,
	)
	return scanStep(row)
}

const getPlaybookSQL = `
SELECT id, name, description, case_type, is_active, org_id, created_at, updated_at
FROM playbooks WHERE id = $1 AND org_id = $2
`

func (r *Repository) GetByID(ctx context.Context, id, orgID uuid.UUID) (playbookRow, error) {
	row := r.db.QueryRowContext(ctx, getPlaybookSQL, id, orgID)
	return scanPlaybook(row)
}

const listPlaybooksSQL = `
SELECT id, name, description, case_type, is_active, org_id, created_at, updated_at
FROM playbooks WHERE org_id = $1
ORDER BY case_type, name
`

func (r *Repository) List(ctx context.Context, orgID uuid.UUID) ([]playbookRow, error) {
	rows, err := r.db.QueryContext(ctx, listPlaybooksSQL, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []playbookRow
	for rows.Next() {
		pb, err := scanPlaybook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pb)
	}
	return out, rows.Err()
}

const listStepsSQL = `
SELECT id, playbook_id, title, description, priority, step_order, due_hours, depends_on_step, created_at
FROM playbook_steps WHERE playbook_id = $1 ORDER BY step_order
`

func (r *Repository) ListSteps(ctx context.Context, playbookID uuid.UUID) ([]stepRow, error) {
	rows, err := r.db.QueryContext(ctx, listStepsSQL, playbookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []stepRow
	for rows.Next() {
		s, err := scanStep(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

const updatePlaybookSQL = `
UPDATE playbooks
SET name        = COALESCE($1, name),
    description = COALESCE($2, description),
    is_active   = COALESCE($3, is_active),
    updated_at  = NOW()
WHERE id = $4 AND org_id = $5
RETURNING id, name, description, case_type, is_active, org_id, created_at, updated_at
`

func (r *Repository) Update(ctx context.Context, id, orgID uuid.UUID, req UpdatePlaybookRequest) (playbookRow, error) {
	var name, description sql.NullString
	var isActive sql.NullBool
	if req.Name != nil {
		name = sql.NullString{String: *req.Name, Valid: true}
	}
	if req.Description != nil {
		description = sql.NullString{String: *req.Description, Valid: true}
	}
	if req.IsActive != nil {
		isActive = sql.NullBool{Bool: *req.IsActive, Valid: true}
	}
	row := r.db.QueryRowContext(ctx, updatePlaybookSQL, name, description, isActive, id, orgID)
	return scanPlaybook(row)
}

const deleteStepsSQL = `DELETE FROM playbook_steps WHERE playbook_id = $1`
const softDeletePlaybookSQL = `UPDATE playbooks SET is_active = false, updated_at = NOW() WHERE id = $1 AND org_id = $2`

func (r *Repository) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, softDeletePlaybookSQL, id, orgID)
	return err
}

func (r *Repository) DeleteSteps(ctx context.Context, playbookID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, deleteStepsSQL, playbookID)
	return err
}

// FindActiveByType returns the first active playbook for a given case_type + org.
const findActiveByTypeSQL = `
SELECT id, name, description, case_type, is_active, org_id, created_at, updated_at
FROM playbooks WHERE case_type = $1 AND org_id = $2 AND is_active = true
ORDER BY created_at LIMIT 1
`

func (r *Repository) FindActiveByType(ctx context.Context, caseType string, orgID uuid.UUID) (*playbookRow, error) {
	row := r.db.QueryRowContext(ctx, findActiveByTypeSQL, caseType, orgID)
	pb, err := scanPlaybook(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pb, nil
}

// ── Scanner helpers ───────────────────────────────────────────────────────────

type scannable interface {
	Scan(dest ...any) error
}

func scanPlaybook(row scannable) (playbookRow, error) {
	var p playbookRow
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.CaseType, &p.IsActive, &p.OrgID, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func scanStep(row scannable) (stepRow, error) {
	var s stepRow
	var priorityStr string
	err := row.Scan(&s.ID, &s.PlaybookID, &s.Title, &s.Description, &priorityStr, &s.StepOrder, &s.DueHours, &s.DependsOnStep, &s.CreatedAt)
	if err == nil {
		s.Priority = db.TaskPriority(priorityStr)
	}
	return s, err
}
