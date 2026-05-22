// Package worker provides a bounded goroutine pool for fire-and-forget tasks.
//
// The previous pattern of `go fn()` scattered through service code spawns an
// unbounded number of goroutines under load, each holding a DB connection for
// its duration, exhausting the pgx connection pool. This package replaces that
// pattern with a fixed-size worker pool backed by a buffered channel.
//
// Usage:
//
//	worker.Background.Submit(func() { repo.CreateActivity(ctx, ...) })
package worker

import (
	"go.uber.org/zap"
	"sync"
)

// Pool is a bounded goroutine pool. At most Workers goroutines run concurrently;
// at most QueueDepth jobs wait. When the queue is full, Submit returns false and
// the caller should log a warning — the job is dropped rather than a new goroutine
// being spawned, which preserves the DB connection pool budget.
type Pool struct {
	jobs     chan func()
	wg       sync.WaitGroup
	stopOnce sync.Once
	logger   *zap.Logger
}

// New creates a Pool with the given number of worker goroutines and job queue depth.
// Workers start immediately and drain jobs until Stop is called.
func New(workers, queueDepth int, logger *zap.Logger) *Pool {
	p := &Pool{
		jobs:   make(chan func(), queueDepth),
		logger: logger,
	}
	for i := 0; i < workers; i++ {
		p.wg.Add(1)
		go p.run()
	}
	return p
}

func (p *Pool) run() {
	defer p.wg.Done()
	for fn := range p.jobs {
		p.safeCall(fn)
	}
}

func (p *Pool) safeCall(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			if p.logger != nil {
				p.logger.Error("worker pool: recovered panic in background job", zap.Any("panic", r))
			}
		}
	}()
	fn()
}

// Submit enqueues fn for asynchronous execution. Returns true if enqueued,
// false if the queue is full (job dropped — caller should log at warn level).
func (p *Pool) Submit(fn func()) bool {
	select {
	case p.jobs <- fn:
		return true
	default:
		if p.logger != nil {
			p.logger.Warn("worker pool: queue full, background job dropped")
		}
		return false
	}
}

// Stop closes the job channel and waits for all in-flight jobs to complete.
// Call this during graceful shutdown. Idempotent — safe to call multiple times.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		close(p.jobs)
	})
	p.wg.Wait()
}

// Background is the shared pool for service-layer fire-and-forget operations
// (activity writes, audit log writes, timeline emissions).
// 8 workers × 512-deep queue: handles ≈8 concurrent DB writes with a burst buffer
// of 512 pending jobs before back-pressure kicks in.
// Initialised by the server on startup via Init; nil-safe (Submit is a no-op when nil).
var Background *Pool

// Init initialises the shared Background pool. Call once from server startup.
func Init(logger *zap.Logger) {
	Background = New(8, 512, logger)
}

// TrySubmit submits fn to the Background pool. If Background is nil (tests that
// don't call Init), fn is executed synchronously so tests always observe the side effect.
func TrySubmit(fn func()) {
	if Background == nil {
		fn()
		return
	}
	Background.Submit(fn)
}
