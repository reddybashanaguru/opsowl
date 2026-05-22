package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID stamps every request with a UUID request ID. If the caller already
// provides X-Request-Id, that value is reused (supports distributed tracing).
// The ID is placed in the context (use RequestIDFromCtx to retrieve it) and
// echoed in the X-Request-Id response header so clients can correlate logs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromCtx retrieves the request ID from a context set by RequestID middleware.
func RequestIDFromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// AccessLog writes a structured zap log entry for every HTTP request.
// Fields: method, path, status, duration_ms, request_id, user_id.
// Health/metrics endpoints are skipped to avoid log noise.
func AccessLog(logger *zap.Logger, jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rec.status),
				zap.String("request_id", RequestIDFromCtx(r.Context())),
				zap.String("remote_ip", realIPFromRequest(r)),
			}

			// Attach user_id when a valid JWT is present.
			if tok := extractBearer(r); tok != "" {
				if claims, err := jwtMgr.Verify(tok); err == nil {
					fields = append(fields, zap.String("user_id", claims.UserID.String()))
				}
			}

			if rec.status >= 500 {
				logger.Error("request completed", fields...)
			} else if rec.status >= 400 {
				logger.Warn("request completed", fields...)
			} else {
				logger.Info("request completed", fields...)
			}
		})
	}
}
