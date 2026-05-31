package alerts

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	// Webhook ingest — no JWT, authenticated via HMAC-SHA256 signature
	r.Post("/alerts/ingest/{sourceID}", h.IngestWebhook)

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Manual ingest from UI (analyst submits raw payload)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/alerts/manual/{sourceID}", h.ManualIngest)

		// Alert queue — list and act on incoming alerts
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/alerts", h.List)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/alerts/{id}", h.Get)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/alerts/{id}/convert", h.Convert)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/alerts/{id}/dismiss", h.Dismiss)

		// Alert sources — admin configures TMS connections
		r.With(auth.RequireRole("admin", "super_admin")).
			Get("/alert-sources", h.ListSources)
		r.With(auth.RequireRole("admin", "super_admin")).
			Post("/alert-sources", h.CreateSource)
		r.With(auth.RequireRole("admin", "super_admin")).
			Get("/alert-sources/{id}", h.GetSource)
		r.With(auth.RequireRole("admin", "super_admin")).
			Patch("/alert-sources/{id}", h.UpdateSource)
		r.With(auth.RequireRole("admin", "super_admin")).
			Delete("/alert-sources/{id}", h.DeleteSource)

		// Alert routing rules — admin configures routing logic
		r.With(auth.RequireRole("admin", "super_admin")).
			Get("/alert-rules", h.ListRules)
		r.With(auth.RequireRole("admin", "super_admin")).
			Post("/alert-rules", h.CreateRule)
		r.With(auth.RequireRole("admin", "super_admin")).
			Patch("/alert-rules/{id}", h.UpdateRule)
		r.With(auth.RequireRole("admin", "super_admin")).
			Delete("/alert-rules/{id}", h.DeleteRule)
	})
}
