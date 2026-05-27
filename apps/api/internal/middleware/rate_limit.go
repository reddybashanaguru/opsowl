package middleware

import (
	"container/list"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// maxBuckets is the maximum number of per-user/IP token buckets held in memory.
// The LRU evicts the least-recently-seen entry when this limit is reached,
// bounding memory at ~(maxBuckets × ~120 bytes) ≈ 6 MB at 50 000 entries.
const maxBuckets = 50_000

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// lruBuckets is a thread-safe LRU map of token buckets.
// It replaces the previous unbounded map + 5-minute cleanup goroutine, which
// held a global mutex during cleanup and could pause all requests for ~100ms.
type lruBuckets struct {
	mu      sync.Mutex
	items   map[string]*list.Element
	order   *list.List // front = most-recently used
	maxSize int
}

type lruEntry struct {
	key string
	b   bucket
}

func newLRUBuckets(maxSize int) *lruBuckets {
	return &lruBuckets{
		items:   make(map[string]*list.Element, maxSize),
		order:   list.New(),
		maxSize: maxSize,
	}
}

// get returns the bucket for key and moves it to the front.
// Returns (zero-value, false) if not present.
func (l *lruBuckets) get(key string) (bucket, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	el, ok := l.items[key]
	if !ok {
		return bucket{}, false
	}
	l.order.MoveToFront(el)
	return el.Value.(*lruEntry).b, true
}

// set upserts a bucket for key, evicting the LRU entry if at capacity.
func (l *lruBuckets) set(key string, b bucket) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[key]; ok {
		l.order.MoveToFront(el)
		el.Value.(*lruEntry).b = b
		return
	}
	if l.order.Len() >= l.maxSize {
		// Evict least-recently-used (back of list).
		oldest := l.order.Back()
		if oldest != nil {
			l.order.Remove(oldest)
			delete(l.items, oldest.Value.(*lruEntry).key)
		}
	}
	el := l.order.PushFront(&lruEntry{key: key, b: b})
	l.items[key] = el
}

// Len returns the current number of tracked buckets.
func (l *lruBuckets) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.order.Len()
}

// RateLimiter is a per-user/IP token-bucket rate limiter backed by a bounded
// LRU map. Memory is capped at maxBuckets entries with no background goroutine.
type RateLimiter struct {
	buckets *lruBuckets
	rate    float64 // tokens per second
	burst   float64
	jwtMgr  *auth.JWTManager
}

func NewRateLimiter(requestsPerMinute int, jwtMgr *auth.JWTManager) *RateLimiter {
	return &RateLimiter{
		buckets: newLRUBuckets(maxBuckets),
		rate:    float64(requestsPerMinute) / 60.0,
		burst:   float64(requestsPerMinute),
		jwtMgr:  jwtMgr,
	}
}

// NewIPRateLimiter creates a rate limiter keyed strictly by client IP.
// Use this on pre-authentication endpoints (login, refresh, MFA challenge)
// where no JWT is present yet. Default burst = requestsPerMinute.
func NewIPRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		buckets: newLRUBuckets(maxBuckets),
		rate:    float64(requestsPerMinute) / 60.0,
		burst:   float64(requestsPerMinute),
		jwtMgr:  nil, // nil → always key by IP (see rateLimitKey)
	}
}

func (rl *RateLimiter) allow(key string) (bool, float64) {
	now := time.Now()

	b, ok := rl.buckets.get(key)
	if !ok {
		b = bucket{tokens: rl.burst, lastSeen: now}
	}

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		rl.buckets.set(key, b)
		wait := (1 - b.tokens) / rl.rate
		return false, wait
	}
	b.tokens--
	rl.buckets.set(key, b)
	return true, 0
}

// rateLimitKey returns a per-user key when a valid JWT is present,
// falling back to IP address. This prevents all users behind a corporate
// proxy from sharing a single bucket (common in Indian enterprise networks).
func (rl *RateLimiter) rateLimitKey(r *http.Request) string {
	if rl.jwtMgr != nil {
		if tok := extractBearer(r); tok != "" {
			if claims, err := rl.jwtMgr.Verify(tok); err == nil {
				return "user:" + claims.UserID.String()
			}
		}
	}
	return "ip:" + realIPFromRequest(r)
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := rl.rateLimitKey(r)
		ok, retryAfter := rl.allow(key)
		if !ok {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			response.Err(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests — please slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func realIPFromRequest(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return r.RemoteAddr
}
