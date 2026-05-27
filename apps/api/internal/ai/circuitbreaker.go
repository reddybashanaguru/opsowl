package ai

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when all requests are being shed due to repeated failures.
var ErrCircuitOpen = errors.New("AI service circuit breaker open — service is temporarily unavailable")

type cbState int

const (
	cbClosed   cbState = iota // normal — requests pass through
	cbOpen                    // failing — requests rejected immediately
	cbHalfOpen                // recovering — one probe request allowed
)

// circuitBreaker is a simple three-state circuit breaker for the Python AI service.
// After failureThreshold consecutive failures the circuit opens; after openTimeout
// it moves to half-open and allows one probe. A successful probe closes the circuit;
// a failed probe resets the open timer.
type circuitBreaker struct {
	mu               sync.Mutex
	state            cbState
	failures         int
	failureThreshold int
	openTimeout      time.Duration
	openedAt         time.Time
}

func newCircuitBreaker(failureThreshold int, openTimeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
	}
}

// allow returns true when the request should be forwarded to the AI service.
func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		if time.Since(cb.openedAt) >= cb.openTimeout {
			cb.state = cbHalfOpen
			return true // allow one probe
		}
		return false
	case cbHalfOpen:
		return false // only one concurrent probe at a time
	}
	return false
}

// recordSuccess transitions open/half-open → closed and resets the failure counter.
func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = cbClosed
	cb.failures = 0
}

// recordFailure increments failure count and opens the circuit when the threshold is reached.
func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.state == cbHalfOpen || cb.failures >= cb.failureThreshold {
		cb.state = cbOpen
		cb.openedAt = time.Now()
	}
}
