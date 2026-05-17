package notes

import (
	"encoding/json"
	"sync"
)

// NoteEvent is the JSON payload pushed over SSE to case viewers.
type NoteEvent struct {
	Type    string `json:"type"` // "note_added" | "comment_added"
	CaseID  string `json:"case_id"`
	Payload any    `json:"payload"`
}

// Broker fans out NoteEvents to all SSE clients watching a case.
type Broker struct {
	mu      sync.Mutex
	clients map[string][]chan string // caseID → slice of client channels
}

func NewBroker() *Broker {
	return &Broker{clients: make(map[string][]chan string)}
}

// Subscribe returns a channel and a deregister func.
func (b *Broker) Subscribe(caseID string) (chan string, func()) {
	ch := make(chan string, 32)
	b.mu.Lock()
	b.clients[caseID] = append(b.clients[caseID], ch)
	b.mu.Unlock()
	return ch, func() { b.deregister(caseID, ch) }
}

func (b *Broker) deregister(caseID string, target chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.clients[caseID]
	updated := list[:0]
	for _, ch := range list {
		if ch != target {
			updated = append(updated, ch)
		}
	}
	if len(updated) == 0 {
		delete(b.clients, caseID)
	} else {
		b.clients[caseID] = updated
	}
}

// Publish sends the event JSON to all subscribers for the case.
func (b *Broker) Publish(evt NoteEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	b.mu.Lock()
	list := make([]chan string, len(b.clients[evt.CaseID]))
	copy(list, b.clients[evt.CaseID])
	b.mu.Unlock()

	for _, ch := range list {
		select {
		case ch <- string(data):
		default:
			// drop slow consumers; client will refetch on reconnect
		}
	}
}
