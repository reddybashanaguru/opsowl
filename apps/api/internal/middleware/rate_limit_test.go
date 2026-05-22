package middleware

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ── lruBuckets unit tests ────────────────────────────────────────────────────

func TestLRUBuckets_SetGet(t *testing.T) {
	l := newLRUBuckets(10)
	b := bucket{tokens: 5, lastSeen: time.Now()}
	l.set("key1", b)

	got, ok := l.get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if got.tokens != 5 {
		t.Fatalf("want tokens=5, got %v", got.tokens)
	}
}

func TestLRUBuckets_MissReturnsNotOK(t *testing.T) {
	l := newLRUBuckets(10)
	_, ok := l.get("nonexistent")
	if ok {
		t.Fatal("expected false for missing key")
	}
}

func TestLRUBuckets_EvictsLRUWhenFull(t *testing.T) {
	const size = 5
	l := newLRUBuckets(size)

	for i := 0; i < size; i++ {
		l.set(fmt.Sprintf("key%d", i), bucket{tokens: float64(i)})
	}
	if l.Len() != size {
		t.Fatalf("expected %d entries, got %d", size, l.Len())
	}

	// Access key0..key3 to make key4 the LRU (it was inserted first and never touched).
	// Actually insertion order: key0 (oldest) ... key4 (newest).
	// key0 is LRU. Access key1..key4 so key0 remains LRU.
	for i := 1; i < size; i++ {
		l.get(fmt.Sprintf("key%d", i))
	}

	// Insert one more — should evict key0.
	l.set("new", bucket{tokens: 99})
	if l.Len() != size {
		t.Fatalf("expected %d entries after eviction, got %d", size, l.Len())
	}
	if _, ok := l.get("key0"); ok {
		t.Fatal("expected key0 to be evicted (LRU)")
	}
	if _, ok := l.get("new"); !ok {
		t.Fatal("expected new key to exist after insertion")
	}
}

func TestLRUBuckets_UpdateExistingKeyDoesNotGrow(t *testing.T) {
	l := newLRUBuckets(3)
	l.set("k", bucket{tokens: 1})
	l.set("k", bucket{tokens: 2})
	l.set("k", bucket{tokens: 3})
	if l.Len() != 1 {
		t.Fatalf("expected 1 entry, got %d", l.Len())
	}
	b, _ := l.get("k")
	if b.tokens != 3 {
		t.Fatalf("want 3, got %v", b.tokens)
	}
}

func TestLRUBuckets_NeverExceedsMaxSize(t *testing.T) {
	const max = 100
	l := newLRUBuckets(max)
	for i := 0; i < max*3; i++ {
		l.set(fmt.Sprintf("key%d", i), bucket{tokens: 1})
	}
	if l.Len() > max {
		t.Fatalf("LRU grew beyond max: %d > %d", l.Len(), max)
	}
}

func TestLRUBuckets_ConcurrentAccess(t *testing.T) {
	l := newLRUBuckets(100)
	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n%50)
			l.set(key, bucket{tokens: float64(n), lastSeen: time.Now()})
			l.get(key)
		}(i)
	}
	wg.Wait()
	if l.Len() > 100 {
		t.Fatalf("LRU grew beyond max under concurrent access: %d", l.Len())
	}
}

// ── RateLimiter allow() tests ────────────────────────────────────────────────

func TestRateLimiter_AllowsUpToBurst(t *testing.T) {
	rl := NewRateLimiter(60, nil) // 1 token/sec, burst=60
	for i := 0; i < 60; i++ {
		ok, _ := rl.allow("user:test")
		if !ok {
			t.Fatalf("expected allow on request %d", i+1)
		}
	}
	// 61st request should be denied
	ok, wait := rl.allow("user:test")
	if ok {
		t.Fatal("expected deny after burst exhausted")
	}
	if wait <= 0 {
		t.Fatalf("expected positive retry-after, got %v", wait)
	}
}

func TestRateLimiter_RefillsOverTime(t *testing.T) {
	rl := NewRateLimiter(60, nil) // 1 token/sec

	// Drain entirely
	for i := 0; i < 60; i++ {
		rl.allow("user:refill")
	}
	ok, _ := rl.allow("user:refill")
	if ok {
		t.Fatal("bucket should be empty")
	}

	// Manually advance lastSeen by injecting a bucket with older timestamp.
	// We test the refill logic by using a high rate so a very short sleep suffices.
	rl2 := NewRateLimiter(600, nil) // 10 tokens/sec
	for i := 0; i < 600; i++ {
		rl2.allow("user:refill2")
	}
	// After ~100ms, ~1 token should be available
	time.Sleep(110 * time.Millisecond)
	ok, _ = rl2.allow("user:refill2")
	if !ok {
		t.Fatal("expected at least 1 token to refill after 110ms at 10 tokens/sec")
	}
}

func TestRateLimiter_IndependentBucketsPerKey(t *testing.T) {
	rl := NewRateLimiter(2, nil) // burst=2 per key
	ok1, _ := rl.allow("user:a")
	ok2, _ := rl.allow("user:a")
	ok3, _ := rl.allow("user:a") // should fail
	ok4, _ := rl.allow("user:b") // different key — should pass

	if !ok1 || !ok2 {
		t.Fatal("first two requests for user:a should be allowed")
	}
	if ok3 {
		t.Fatal("third request for user:a should be denied")
	}
	if !ok4 {
		t.Fatal("user:b should have its own full bucket")
	}
}

func TestRateLimiter_BucketCountBounded(t *testing.T) {
	rl := NewRateLimiter(100, nil)
	for i := 0; i < maxBuckets*2; i++ {
		rl.allow(fmt.Sprintf("ip:192.168.1.%d", i))
	}
	n := rl.buckets.Len()
	if n > maxBuckets {
		t.Fatalf("bucket count %d exceeds maxBuckets %d", n, maxBuckets)
	}
}
