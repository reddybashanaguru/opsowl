package timeline

import (
	"sync"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// Client represents a single SSE connection for a specific case.
type Client struct {
	CaseID string
	Ch     chan db.TimelineEvent
}

// Hub manages all active SSE clients, keyed by case ID.
type Hub struct {
	mu      sync.Mutex
	clients map[string][]*Client
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string][]*Client)}
}

// Subscribe registers a client for a case and returns a cleanup func.
func (h *Hub) Subscribe(caseID string) (*Client, func()) {
	c := &Client{CaseID: caseID, Ch: make(chan db.TimelineEvent, 64)}
	h.mu.Lock()
	h.clients[caseID] = append(h.clients[caseID], c)
	h.mu.Unlock()
	return c, func() { h.deregister(caseID, c) }
}

func (h *Hub) deregister(caseID string, target *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.clients[caseID]
	updated := list[:0]
	for _, c := range list {
		if c != target {
			updated = append(updated, c)
		}
	}
	if len(updated) == 0 {
		delete(h.clients, caseID)
	} else {
		h.clients[caseID] = updated
	}
}

// Publish sends an event to all clients subscribed to the given case.
func (h *Hub) Publish(caseID string, evt db.TimelineEvent) {
	h.mu.Lock()
	list := make([]*Client, len(h.clients[caseID]))
	copy(list, h.clients[caseID])
	h.mu.Unlock()

	for _, c := range list {
		select {
		case c.Ch <- evt:
		default: // client buffer full — skip; client replays on reconnect
		}
	}
}
