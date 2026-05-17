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

var uuidPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// AuditLog writes a row to audit_logs for every non-GET, non-health request.
// It parses the bearer token directly so it works regardless of JWT middleware ordering.
func AuditLog(q *db.Queries, jwtMgr *auth.JWTManager, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Skip read-only methods and health endpoints.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				return
			}
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
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

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := q.CreateAuditLog(ctx, db.CreateAuditLogParams{
					EventCode: eventCode,
					ActorID:   actorID,
					IpAddress: sql.NullString{String: r.RemoteAddr, Valid: r.RemoteAddr != ""},
					Metadata:  json.RawMessage(meta),
					Severity:  severity,
				}); err != nil {
					logger.Warn("audit log write failed", zap.Error(err))
				}
			}()
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
