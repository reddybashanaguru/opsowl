package timeline

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Emitter is the interface every module uses to write timeline events.
type Emitter interface {
	Emit(ctx context.Context, evt Event) error
}

// WebhookDispatcher is the narrow interface DBEmitter uses to fan out events
// to outbound webhooks without importing the webhooks package.
type WebhookDispatcher interface {
	Dispatch(ctx context.Context, eventType string, caseID *uuid.UUID, payload map[string]any)
}

// DBEmitter persists events to the DB and fans out to SSE and optional webhooks.
type DBEmitter struct {
	q        *db.Queries
	hub      *Hub
	webhooks WebhookDispatcher
}

func NewEmitter(q *db.Queries, hub *Hub) *DBEmitter {
	return &DBEmitter{q: q, hub: hub}
}

// WithWebhooks attaches an outbound webhook dispatcher. Returns the same
// *DBEmitter so the caller can chain: timeline.NewEmitter(...).WithWebhooks(svc).
func (e *DBEmitter) WithWebhooks(d WebhookDispatcher) *DBEmitter {
	e.webhooks = d
	return e
}

func (e *DBEmitter) Emit(ctx context.Context, evt Event) error {
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

	// Fan-out to SSE clients
	e.hub.Publish(evt.CaseID.String(), row)

	// Fan-out to outbound webhooks (non-blocking)
	if e.webhooks != nil {
		caseID := evt.CaseID
		go e.webhooks.Dispatch(ctx, string(evt.EventType), &caseID, evt.Payload)
	}

	return nil
}

// NoopEmitter satisfies the Emitter interface without doing anything.
// Useful in tests that don't care about timeline side-effects.
type NoopEmitter struct{}

func (NoopEmitter) Emit(_ context.Context, _ Event) error { return nil }

func marshalJSON(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}
