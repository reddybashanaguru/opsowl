package cases

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type CreateCaseRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Priority    db.CasePriority `json:"priority"`
	CaseType    string          `json:"case_type,omitempty"`
	AssignedTo  *uuid.UUID      `json:"assigned_to,omitempty"`
	DueDate     *time.Time      `json:"due_date,omitempty"`
	Tags        []string        `json:"tags"`
}

type UpdateCaseRequest struct {
	Title       *string          `json:"title,omitempty"`
	Description *string          `json:"description,omitempty"`
	Priority    *db.CasePriority `json:"priority,omitempty"`
	DueDate     *time.Time       `json:"due_date,omitempty"`
}

type StatusRequest struct {
	Status db.CaseStatus `json:"status"`
	Reason string        `json:"reason"`
}

type AssignRequest struct {
	UserID uuid.UUID `json:"assigned_to"`
	Note   string    `json:"note"`
}

type TagsRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

type UserRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type CaseResponse struct {
	ID             uuid.UUID       `json:"id"`
	CaseNumber     string          `json:"case_number"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Status         db.CaseStatus   `json:"status"`
	Priority       db.CasePriority `json:"priority"`
	CaseType       string          `json:"case_type,omitempty"`
	AssignedTo     *UserRef        `json:"assigned_to"`
	CreatedBy      UserRef         `json:"created_by"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	DueDate        *time.Time      `json:"due_date"`
	SLABreachAt    *time.Time      `json:"sla_breach_at"`
	SLABreached    bool            `json:"sla_breached"`
	Tags           []string        `json:"tags"`
	Watchers       []UserRef       `json:"watchers"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	RiskScore      *string         `json:"risk_score,omitempty"`
	RiskLevel      *string         `json:"risk_level,omitempty"`
	OpenTasksCount int             `json:"open_tasks_count,omitempty"`
	DocumentsCount int             `json:"documents_count,omitempty"`
}

type ActivityResponse struct {
	ID        uuid.UUID              `json:"id"`
	EventType string                 `json:"event_type"`
	Actor     UserRef                `json:"actor"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"created_at"`
}

type ListMeta struct {
	NextCursor *uuid.UUID `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
	TotalCount int64      `json:"total_count"`
}

type ListFilters struct {
	Status      *db.CaseStatus
	Priority    *db.CasePriority
	AssignedTo  *uuid.UUID
	SLABreached *bool
	From        *time.Time
	To          *time.Time
	Cursor      *uuid.UUID
	Limit       int32
	Q           *string
	// ViewerID restricts results to cases accessible to this user (analyst/viewer access control).
	// Non-nil only for analyst and viewer roles.
	ViewerID *uuid.UUID
	// ActiveOnly excludes draft and closed statuses — used by the "Active Cases" dashboard filter.
	ActiveOnly bool
	// Statuses filters to cases matching any of the listed statuses (multi-select).
	Statuses []db.CaseStatus
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func ptrTime(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}
