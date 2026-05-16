package middleware

import (
	"context"
	"net/http"
	"time"
)

// AuditEvent is written asynchronously after each request on protected routes.
type AuditEvent struct {
	EventCode    string
	ActorID      string
	IPAddress    string
	ResourceType string
	ResourceID   string
	Metadata     map[string]any
	Severity     string
	Timestamp    time.Time
}

// AuditWriter persists audit events. Implemented by the auth repository in production,
// swapped for a no-op in tests.
type AuditWriter interface {
	WriteEvent(ctx context.Context, e AuditEvent)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// AuditIntercept wraps responses to capture status codes for audit logging.
// Full audit writing is handled by feature-level middleware; this captures HTTP outcomes.
func AuditIntercept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		// Status available as rec.status for downstream audit hooks.
	})
}
