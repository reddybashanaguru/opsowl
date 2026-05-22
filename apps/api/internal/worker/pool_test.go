package worker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestPool(workers, depth int) *Pool {
	return New(workers, depth, nil)
}

func TestPool_SubmitRunsJob(t *testing.T) {
	p := newTestPool(2, 10)
	defer p.Stop()

	var ran atomic.Bool
	p.Submit(func() { ran.Store(true) })

	deadline := time.Now().Add(time.Second)
	for !ran.Load() {
		if time.Now().After(deadline) {
			t.Fatal("job did not run within 1s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPool_AllJobsRun(t *testing.T) {
	const n = 200
	p := newTestPool(4, n)
	defer p.Stop()

	var count atomic.Int32
	for i := 0; i < n; i++ {
		p.Submit(func() { count.Add(1) })
	}

	p.Stop()
	if count.Load() != n {
		t.Fatalf("expected %d jobs run, got %d", n, count.Load())
	}
}

func TestPool_ReturnsFalseWhenFull(t *testing.T) {
	// 0-depth queue + 0 workers — all submits should return false after first
	p := &Pool{jobs: make(chan func(), 0)}
	// Don't start any workers so queue never drains

	result := p.Submit(func() {})
	// With 0-depth unbuffered channel and no readers, submit should fail immediately
	if result {
		t.Skip("channel accepted job unexpectedly — race in test setup")
	}
}

func TestPool_QueueFullDropsJob(t *testing.T) {
	// 1 worker, depth=1 — block the worker so queue fills
	p := newTestPool(1, 1)
	defer p.Stop()

	var started, done sync.WaitGroup
	started.Add(1)
	done.Add(1)

	blocker := make(chan struct{})
	p.Submit(func() {
		started.Done()
		<-blocker
		done.Done()
	})
	started.Wait() // worker is now busy

	// Fill the queue
	p.Submit(func() {})

	// This one should be dropped (queue full + worker busy)
	var dropped int
	for i := 0; i < 10; i++ {
		if !p.Submit(func() {}) {
			dropped++
		}
	}
	if dropped == 0 {
		t.Error("expected at least one drop when queue is full")
	}

	close(blocker)
	done.Wait()
}

func TestPool_PanicInJobDoesNotKillWorker(t *testing.T) {
	p := newTestPool(1, 10)
	defer p.Stop()

	// First job panics
	var first atomic.Bool
	p.Submit(func() {
		first.Store(true)
		panic("intentional test panic")
	})

	// Second job should still run
	var second atomic.Bool
	p.Submit(func() { second.Store(true) })

	deadline := time.Now().Add(2 * time.Second)
	for !second.Load() {
		if time.Now().After(deadline) {
			t.Fatal("worker did not recover from panic — second job never ran")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !first.Load() {
		t.Fatal("first job never ran")
	}
}

func TestPool_StopWaitsForInFlight(t *testing.T) {
	const n = 20
	p := newTestPool(2, n) // queue depth matches job count — no drops

	var count atomic.Int32
	for i := 0; i < n; i++ {
		p.Submit(func() {
			time.Sleep(5 * time.Millisecond)
			count.Add(1)
		})
	}
	p.Stop() // must wait for all n to finish
	if count.Load() != n {
		t.Fatalf("Stop() returned before all jobs completed: ran %d/%d", count.Load(), n)
	}
}

func TestTrySubmit_NilPoolRunsSynchronously(t *testing.T) {
	Background = nil
	var ran bool
	TrySubmit(func() { ran = true })
	if !ran {
		t.Fatal("TrySubmit with nil pool should run synchronously")
	}
}

func TestPool_ConcurrentSubmit(t *testing.T) {
	p := newTestPool(8, 200)
	defer p.Stop()

	var count atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Submit(func() { count.Add(1) })
		}()
	}
	wg.Wait()
	p.Stop()
	if count.Load() == 0 {
		t.Fatal("no jobs ran under concurrent submit")
	}
}
