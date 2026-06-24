package ai

import (
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	// AI endpoints can be slow — override the global 30s timeout with 120s.
	aiTimeout := chimw.Timeout(120 * time.Second)

	// Agent streaming: JWT auth but no timeout — runs until done or client disconnects.
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/ai/agent", h.RunAgent)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Use(aiTimeout)

		// ── Per-case copilot actions ──────────────────────────────────────
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/ai/ask", h.Ask)

		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/ai/summarize", h.Summarize)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/ai/generate-note", h.GenerateNote)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/ai/extract-actions", h.ExtractActions)

		// ── AI log reads ──────────────────────────────────────────────────
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{caseId}/ai/logs", h.ListLogs)

		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/ai/logs/{logId}", h.GetLog)

		// ── Human review actions ──────────────────────────────────────────
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/ai/logs/{logId}/approve", h.Approve)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/ai/logs/{logId}/reject", h.Reject)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Put("/ai/logs/{logId}/edit", h.Edit)

		// ── Pending review queue (managers only) ──────────────────────────
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/ai/logs/pending", h.ListPending)

		// ── EU AI Act audit trail CSV export (manager+) ───────────────────
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/ai/logs/export", h.ExportCSV)
	})
}
