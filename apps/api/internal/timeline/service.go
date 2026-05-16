package timeline

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var ErrNotFound = errors.New("timeline event not found")

// userEmittableTypes — only these event types may be created via POST
var userEmittableTypes = map[db.TimelineEventType]bool{
	db.TimelineEventTypeCaseStatusChanged: true,
	db.TimelineEventTypeCaseAssigned:      true,
	db.TimelineEventTypeCaseTagged:        true,
	db.TimelineEventTypeNoteAdded:         true,
	db.TimelineEventTypeUserAction:        true,
}

type Service struct {
	repo    *Repository
	emitter Emitter
}

func NewService(repo *Repository, emitter Emitter) *Service {
	return &Service{repo: repo, emitter: emitter}
}

type cursorPayload struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         uuid.UUID `json:"id"`
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	b, _ := json.Marshal(cursorPayload{OccurredAt: t, ID: id})
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCursor(s string) (*time.Time, *uuid.UUID, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, nil, err
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil, err
	}
	return &p.OccurredAt, &p.ID, nil
}

func (s *Service) List(ctx context.Context, caseID uuid.UUID, f ListFilters) ([]EventResponse, ListMeta, error) {
	rows, err := s.repo.List(ctx, caseID, f)
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

	total, _ := s.repo.Count(ctx, caseID, f)

	var nextCursor *string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		c := encodeCursor(last.OccurredAt, last.ID)
		nextCursor = &c
	}

	out := make([]EventResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, toEventResponse(r))
	}
	return out, ListMeta{TotalCount: total, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *Service) Create(ctx context.Context, claims *auth.Claims, caseID uuid.UUID, req CreateEventRequest) (*EventResponse, error) {
	if !userEmittableTypes[req.EventType] {
		return nil, errors.New("event type cannot be manually created")
	}
	actorID := claims.UserID
	evt := Event{
		CaseID:     caseID,
		EventType:  req.EventType,
		Source:     db.TimelineEventSourceUser,
		ActorID:    &actorID,
		OccurredAt: req.OccurredAt,
		Payload:    req.Payload,
	}
	if err := s.emitter.Emit(ctx, evt); err != nil {
		return nil, err
	}
	return nil, nil // response is minimal for manual events
}

func toEventResponse(r db.ListTimelineEventsRow) EventResponse {
	var payload map[string]any
	_ = json.Unmarshal(r.Payload, &payload)

	resp := EventResponse{
		ID:         r.ID,
		CaseID:     r.CaseID,
		EventType:  r.EventType,
		Source:     r.Source,
		OccurredAt: r.OccurredAt,
		CreatedAt:  r.CreatedAt,
		Payload:    payload,
		BackDated:  r.CreatedAt.Sub(r.OccurredAt) > time.Hour || r.OccurredAt.Sub(r.CreatedAt) > time.Hour,
	}
	if r.ActorID.Valid {
		resp.Actor = &ActorRef{
			ID:       r.ActorID.UUID,
			FullName: r.ActorName.String,
			Email:    r.ActorEmail.String,
		}
	}
	return resp
}
