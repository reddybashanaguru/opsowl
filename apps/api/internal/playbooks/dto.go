package playbooks

import (
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrNotFound      = errors.New("playbook not found")
	ErrForbidden     = errors.New("forbidden")
	ErrNameRequired  = errors.New("playbook name is required")
	ErrCaseTypeRequired = errors.New("case_type is required")
)

// ── Request / Response types ──────────────────────────────────────────────────

type CreatePlaybookRequest struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	CaseType    string               `json:"case_type"`
	Steps       []CreateStepRequest  `json:"steps"`
}

type CreateStepRequest struct {
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Priority       db.TaskPriority `json:"priority"`
	StepOrder      int             `json:"step_order"`
	DueHours       *int            `json:"due_hours"`
	DependsOnStep  int             `json:"depends_on_step"`
}

type UpdatePlaybookRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type StepResponse struct {
	ID             uuid.UUID       `json:"id"`
	PlaybookID     uuid.UUID       `json:"playbook_id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Priority       db.TaskPriority `json:"priority"`
	StepOrder      int             `json:"step_order"`
	DueHours       *int            `json:"due_hours"`
	DependsOnStep  int             `json:"depends_on_step"`
	CreatedAt      time.Time       `json:"created_at"`
}

type PlaybookResponse struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	CaseType    string         `json:"case_type"`
	IsActive    bool           `json:"is_active"`
	OrgID       uuid.UUID      `json:"org_id"`
	Steps       []StepResponse `json:"steps"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ListMeta struct {
	TotalCount int64 `json:"total_count"`
}
