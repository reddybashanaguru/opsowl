package middleware

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

// idempotencyTTL is how long a cached response is retained.
// Matches the expires_at DEFAULT in migration 00022.
const idempotencyTTL = 24 * time.Hour

// bodyRecorder captures both the status code and response body so the full
// response can be stored in the idempotency_keys table for replay.
type bodyRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *bodyRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *bodyRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Flush implements http.Flusher so SSE connections still work through this middleware.
func (r *bodyRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Idempotency protects POST mutations from double-execution under network retries.
//
// If the caller includes an Idempotency-Key header:
//   - On first call: run the handler, cache (key, user_id) → (status, body), return normally.
//   - On retry with same key: return the exact cached response without re-running the handler.
//   - After 24 hours (TTL expires): treat the key as fresh again.
//
// If no Idempotency-Key header is present, the middleware is a transparent no-op.
// Only POST requests are intercepted; GET/PUT/DELETE are passed through unchanged.
//
// Per-user isolation: the same key string from two different users is treated as
// two independent idempotency records.
func Idempotency(db *sql.DB, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only intercept POST mutations with an explicit idempotency key.
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims := auth.ClaimsFromCtx(r.Context())
			if claims == nil {
				// No auth context — pass through (auth middleware will reject it)
				next.ServeHTTP(w, r)
				return
			}
			userID := claims.UserID

			// Check for an existing, non-expired cached response.
			var cachedStatus int
			var cachedBody []byte
			err := db.QueryRowContext(r.Context(),
				`SELECT response_status, response_body
				   FROM idempotency_keys
				  WHERE idem_key = $1
				    AND user_id  = $2
				    AND expires_at > NOW()`,
				key, userID,
			).Scan(&cachedStatus, &cachedBody)

			if err == nil && cachedStatus > 0 {
				// Replay: write the cached Content-Type + body without calling the handler.
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Idempotency-Replayed", "true")
				w.WriteHeader(cachedStatus)
				w.Write(cachedBody) //nolint:errcheck
				return
			}

			// First time this key is seen — run handler and capture response.
			rec := &bodyRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Only cache successful responses (2xx). Don't cache 4xx/5xx so the
			// caller can fix their request and retry without the old error being
			// replayed forever.
			if rec.status >= 200 && rec.status < 300 {
				expires := time.Now().Add(idempotencyTTL)
				_, dbErr := db.ExecContext(context.Background(),
					`INSERT INTO idempotency_keys
					        (idem_key, user_id, method, path, response_status, response_body, expires_at)
					 VALUES ($1, $2, $3, $4, $5, $6, $7)
					 ON CONFLICT (idem_key, user_id) DO NOTHING`,
					key, userID, r.Method, r.URL.Path, rec.status, rec.body.Bytes(), expires,
				)
				if dbErr != nil {
					logger.Warn("idempotency key store failed", zap.Error(dbErr), zap.String("key", key))
				}
			}
		})
	}
}

// IdempotencyCleanup deletes expired idempotency keys. Call periodically
// (e.g. once per day via a cron goroutine or a scheduled worker job).
func IdempotencyCleanup(db *sql.DB, logger *zap.Logger) {
	res, err := db.ExecContext(context.Background(),
		`DELETE FROM idempotency_keys WHERE expires_at < NOW()`,
	)
	if err != nil {
		logger.Warn("idempotency cleanup failed", zap.Error(err))
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		logger.Info("idempotency keys cleaned up", zap.Int64("deleted", n))
	}
}

// idempotencyErrBody is returned when the middleware itself fails after headers sent.
var _ = response.Err // ensure response package is reachable (lint)
