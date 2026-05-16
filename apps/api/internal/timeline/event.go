package timeline

import (
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Event is the domain struct passed to Emit. Modules build this and hand it off.
type Event struct {
	CaseID     uuid.UUID
	EventType  db.TimelineEventType
	Source     db.TimelineEventSource
	ActorID    *uuid.UUID     // nil for system-generated events
	OccurredAt time.Time      // real-world timestamp; can be back-dated
	Payload    map[string]any // serialized to JSONB
	Metadata   map[string]any
}
