package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auditchain"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher so SSE streams work through this middleware.
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// sensitiveReadPaths lists path substrings whose GET access must be audited for RBI compliance.
// Every case, document, note, task, audit log, and AI log read generates a trail entry.
var sensitiveReadPaths = []string{
	"/cases/",
	"/cases?",
	"/documents",
	"/notes",
	"/ai-logs",
	"/audit",
	"/tasks/",
	"/tasks?",
	"/comms",
}

// isSensitiveRead returns true when a GET on this path must be audited.
func isSensitiveRead(path string) bool {
	// Never audit SSE streams — they are long-lived connections that would
	// generate one entry per connection, not per data access.
	if strings.HasSuffix(path, "/stream") {
		return false
	}
	// Skip own profile — not sensitive enough to audit every page load.
	if path == "/api/v1/auth/me" {
		return false
	}
	for _, substr := range sensitiveReadPaths {
		if strings.Contains(path, substr) {
			return true
		}
	}
	return false
}

type auditJob struct {
	params db.CreateAuditLogParams
}

// AuditLog writes a row to audit_logs for every mutating request AND every
// GET on sensitive resources (cases, documents, notes, AI logs, audit viewer).
// Entries are written through ChainWriter which computes a SHA-256 hash chain
// for tamper-evidence (RBI CSF requirement for immutable audit trails).
//
// 4 goroutines drain the channel; ChainWriter serialises them via mutex.
func AuditLog(q *db.Queries, jwtMgr *auth.JWTManager, logger *zap.Logger, chain *auditchain.ChainWriter) func(http.Handler) http.Handler {
	ch := make(chan auditJob, 2048)

	// 4 goroutines drain the channel; ChainWriter.Write() serialises them internally
	// via a mutex to guarantee monotonic seq_num assignment and correct hash linkage.
	for i := 0; i < 4; i++ {
		go func() {
			for job := range ch {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := chain.Write(ctx, job.params); err != nil {
					logger.Warn("audit log write failed", zap.Error(err))
				}
				cancel()
			}
		}()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Skip health and infrastructure endpoints entirely.
			if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/metrics" {
				return
			}
			if r.Method == http.MethodOptions || r.Method == http.MethodHead {
				return
			}

			// For GET: only audit access to sensitive data paths.
			if r.Method == http.MethodGet && !isSensitiveRead(r.URL.Path) {
				return
			}

			// Resolve actor from bearer token (best-effort).
			var actorID uuid.NullUUID
			if tok := extractBearer(r); tok != "" {
				if claims, err := jwtMgr.Verify(tok); err == nil {
					actorID = uuid.NullUUID{UUID: claims.UserID, Valid: true}
				}
			}

			severity := "info"
			if rec.status >= 400 {
				severity = "warn"
			}
			if rec.status >= 500 {
				severity = "error"
			}

			eventCode := buildEventCode(r.Method, r.URL.Path)
			meta, _ := json.Marshal(map[string]any{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rec.status,
				"request_id": r.Header.Get("X-Request-Id"),
			})

			job := auditJob{params: db.CreateAuditLogParams{
				EventCode: eventCode,
				ActorID:   actorID,
				IpAddress: sql.NullString{String: r.RemoteAddr, Valid: r.RemoteAddr != ""},
				Metadata:  json.RawMessage(meta),
				Severity:  severity,
			}}

			// Non-blocking send — the channel buffer absorbs bursts.
			// Warn (never silently drop) when the buffer is exhausted.
			select {
			case ch <- job:
			default:
				logger.Warn("audit log channel full — entry dropped",
					zap.String("event_code", eventCode),
					zap.String("path", r.URL.Path))
			}
		})
	}
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

// buildEventCode converts "POST /api/v1/cases/{uuid}/ai/ask" → "api.cases.ai.ask.post"
func buildEventCode(method, path string) string {
	stripped := uuidPattern.ReplaceAllString(path, "")
	stripped = strings.TrimPrefix(stripped, "/api/v1/")
	parts := strings.FieldsFunc(stripped, func(r rune) bool { return r == '/' })
	code := strings.Join(parts, ".") + "." + strings.ToLower(method)
	return code
}
