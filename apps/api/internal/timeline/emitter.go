package timeline

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Emitter is the interface every module uses to write timeline events.
// Using a context-only (non-tx) approach for MVP — events are written async after commit.
type Emitter interface {
	Emit(ctx context.Context, evt Event) error
}

type dbEmitter struct {
	q   *db.Queries
	hub *Hub
}

func NewEmitter(q *db.Queries, hub *Hub) Emitter {
	return &dbEmitter{q: q, hub: hub}
}

func (e *dbEmitter) Emit(ctx context.Context, evt Event) error {
	payload, err := marshalJSON(evt.Payload)
	if err != nil {
		return err
	}
	meta, err := marshalJSON(evt.Metadata)
	if err != nil {
		return err
	}

	var actorID uuid.NullUUID
	if evt.ActorID != nil {
		actorID = uuid.NullUUID{UUID: *evt.ActorID, Valid: true}
	}

	row, err := e.q.InsertTimelineEvent(ctx, db.InsertTimelineEventParams{
		CaseID:     evt.CaseID,
		EventType:  evt.EventType,
		Source:     evt.Source,
		ActorID:    actorID,
		OccurredAt: evt.OccurredAt,
		Payload:    payload,
		Metadata:   meta,
	})
	if err != nil {
		return err
	}

	// Fan-out to SSE clients after successful DB write
	e.hub.Publish(evt.CaseID.String(), row)
	return nil
}

func marshalJSON(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}
