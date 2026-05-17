package notifications

import (
	"encoding/json"
	"sync"
)

// UserBroker fans out notification events to SSE clients keyed by user ID.
type UserBroker struct {
	mu      sync.Mutex
	clients map[string][]chan string // userID → channels
}

func NewBroker() *UserBroker {
	return &UserBroker{clients: make(map[string][]chan string)}
}

func (b *UserBroker) Subscribe(userID string) (chan string, func()) {
	ch := make(chan string, 32)
	b.mu.Lock()
	b.clients[userID] = append(b.clients[userID], ch)
	b.mu.Unlock()
	return ch, func() { b.deregister(userID, ch) }
}

func (b *UserBroker) deregister(userID string, target chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.clients[userID]
	updated := list[:0]
	for _, ch := range list {
		if ch != target {
			updated = append(updated, ch)
		}
	}
	if len(updated) == 0 {
		delete(b.clients, userID)
	} else {
		b.clients[userID] = updated
	}
}

// Push sends a JSON-serialised payload to all SSE clients for the user.
func (b *UserBroker) Push(userID string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	b.mu.Lock()
	list := make([]chan string, len(b.clients[userID]))
	copy(list, b.clients[userID])
	b.mu.Unlock()

	for _, ch := range list {
		select {
		case ch <- string(data):
		default:
		}
	}
}
