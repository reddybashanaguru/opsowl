package db

import (
	"sync"
	"time"
)

// CircuitBreaker is a simple 3-state (closed / open / half-open) guard for
// the database connection pool. A background goroutine calls Probe on every
// health-check tick; HTTP middleware calls IsOpen before processing a request.
//
// States:
//   - Closed  (failures < threshold): all requests pass through normally.
//   - Open    (failures ≥ threshold): requests are rejected with 503 until
//     the cooldown elapses.
//   - Half-open (cooldown elapsed): circuit resets to closed; the next
//     successful Probe keeps it closed, the next failure re-opens it.
type CircuitBreaker struct {
	mu        sync.Mutex
	failures  int
	openedAt  time.Time
	threshold int
	cooldown  time.Duration
}

// NewCircuitBreaker returns a breaker that opens after [threshold] consecutive
// failed probes and attempts half-open recovery after [cooldown].
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{threshold: threshold, cooldown: cooldown}
}

// Probe records the outcome of a database ping. Call from the background
// health-check goroutine. A nil error counts as success and closes the circuit.
func (cb *CircuitBreaker) Probe(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err == nil {
		cb.failures = 0
		cb.openedAt = time.Time{}
		return
	}
	cb.failures++
	if cb.failures >= cb.threshold && cb.openedAt.IsZero() {
		cb.openedAt = time.Now()
	}
}

// IsOpen returns true when requests should be rejected immediately (503).
// After the cooldown elapses the state resets to closed (half-open probe
// attempt): the next Probe result determines whether it reopens.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.openedAt.IsZero() {
		return false
	}
	if time.Since(cb.openedAt) >= cb.cooldown {
		cb.openedAt = time.Time{}
		cb.failures = 0
		return false
	}
	return true
}
