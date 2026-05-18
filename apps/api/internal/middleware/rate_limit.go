package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
	jwtMgr  *auth.JWTManager
}

func NewRateLimiter(requestsPerMinute int, jwtMgr *auth.JWTManager) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(requestsPerMinute) / 60.0,
		burst:   float64(requestsPerMinute),
		jwtMgr:  jwtMgr,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for key, b := range rl.buckets {
			b.mu.Lock()
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, key)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) allow(key string) (bool, float64) {
	rl.mu.Lock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, lastSeen: time.Now()}
		rl.buckets[key] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed*rl.rate)
	b.lastSeen = now

	if b.tokens < 1 {
		// seconds until next token available
		wait := (1 - b.tokens) / rl.rate
		return false, wait
	}
	b.tokens--
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

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
