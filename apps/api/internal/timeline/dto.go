package timeline

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type ListFilters struct {
	EventType *db.TimelineEventType
	Source    *db.TimelineEventSource
	ActorID   *uuid.UUID
	From      *time.Time
	To        *time.Time
	// Cursor encodes (occurred_at, id) for stable keyset pagination
	CursorTime *time.Time
	CursorID   *uuid.UUID
	Limit      int32
}

type ListMeta struct {
	TotalCount int64      `json:"total_count"`
	HasMore    bool       `json:"has_more"`
	NextCursor *string    `json:"next_cursor"`
}

type ActorRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email,omitempty"`
}

type EventResponse struct {
	ID         uuid.UUID              `json:"id"`
	CaseID     uuid.UUID              `json:"case_id"`
	EventType  db.TimelineEventType   `json:"event_type"`
	Source     db.TimelineEventSource `json:"source"`
	Actor      *ActorRef              `json:"actor"`
	OccurredAt time.Time              `json:"occurred_at"`
	CreatedAt  time.Time              `json:"created_at"`
	Payload    map[string]any         `json:"payload"`
	// BackDated is true when occurred_at differs from created_at by > 1 hour
	BackDated bool `json:"back_dated"`
}

type CreateEventRequest struct {
	EventType  db.TimelineEventType   `json:"event_type"`
	Source     db.TimelineEventSource `json:"source"`
	OccurredAt time.Time              `json:"occurred_at"`
	Payload    map[string]any         `json:"payload"`
}

func nullTimeFromPtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
