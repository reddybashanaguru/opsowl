package tasks

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrNotFound           = errors.New("task not found")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrTitleRequired      = errors.New("title is required")
)

// ── Request / Response types ──────────────────────────────────────────────────

type CreateTaskRequest struct {
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Priority        db.TaskPriority  `json:"priority"`
	DueDate         *time.Time       `json:"due_date"`
	AssignedTo      *uuid.UUID       `json:"assigned_to"`
	TimelineEventID *uuid.UUID       `json:"timeline_event_id"`
	IsAISuggested   bool             `json:"is_ai_suggested"`
}

type UpdateTaskRequest struct {
	Title          *string          `json:"title"`
	Description    *string          `json:"description"`
	Priority       *db.TaskPriority `json:"priority"`
	DueDate        *time.Time       `json:"due_date"`
	ClearDueDate   bool             `json:"clear_due_date"`
	AssignedTo     *uuid.UUID       `json:"assigned_to"`
	ClearAssignedTo bool            `json:"clear_assigned_to"`
	Status         *db.TaskStatus   `json:"status"`
	SortOrder      *int32           `json:"sort_order"`
}

type ListFilters struct {
	Status      *db.TaskStatus
	Priority    *db.TaskPriority
	AssignedTo  *uuid.UUID
	OverdueOnly bool
	Limit       int32
	CursorID    *uuid.UUID
}

type ListMeta struct {
	TotalCount int64      `json:"total_count"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor"`
}

type UserRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type TaskResponse struct {
	ID              uuid.UUID        `json:"id"`
	CaseID          uuid.UUID        `json:"case_id"`
	CaseNumber      string           `json:"case_number,omitempty"`
	CaseTitle       string           `json:"case_title,omitempty"`
	Title           string           `json:"title"`
	Description     string           `json:"description,omitempty"`
	Status          db.TaskStatus    `json:"status"`
	Priority        db.TaskPriority  `json:"priority"`
	IsOverdue       bool             `json:"is_overdue"`
	IsAISuggested   bool             `json:"is_ai_suggested"`
	SortOrder       int32            `json:"sort_order"`
	DueDate         *time.Time       `json:"due_date"`
	CompletedAt     *time.Time       `json:"completed_at"`
	CreatedBy       UserRef          `json:"created_by"`
	AssignedTo      *UserRef         `json:"assigned_to"`
	CommentCount    int64            `json:"comment_count"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CommentResponse struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	Author    UserRef   `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
	Content string `json:"content"`
}

// ── Valid state machine ───────────────────────────────────────────────────────

var validTransitions = map[db.TaskStatus][]db.TaskStatus{
	db.TaskStatusTodo:       {db.TaskStatusInProgress, db.TaskStatusCancelled},
	db.TaskStatusInProgress: {db.TaskStatusDone, db.TaskStatusCancelled},
	db.TaskStatusDone:       {},
	db.TaskStatusCancelled:  {},
}

func validateTransition(from, to db.TaskStatus) error {
	for _, allowed := range validTransitions[from] {
		if allowed == to {
			return nil
		}
	}
	return ErrInvalidTransition
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func timePtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
