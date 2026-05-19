package inbound

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	// ── Webhook endpoint — HMAC-authenticated, no JWT ─────────────────────────
	// This is the URL external systems call to push events. Authentication is
	// done via HMAC signature, not JWT — the caller signs each request body.
	r.Post("/inbound/events", h.ReceiveEvent)

	// ── Admin routes — JWT + admin role required ──────────────────────────────
	r.Route("/inbound", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Use(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin))

		// Source management
		r.Get("/sources", h.ListSources)
		r.Post("/sources", h.CreateSource)
		r.Get("/sources/{sourceID}", h.GetSource)
		r.Put("/sources/{sourceID}", h.UpdateSource)
		r.Delete("/sources/{sourceID}", h.DeleteSource)
		r.Post("/sources/{sourceID}/regenerate-secret", h.RegenerateSecret)

		// Rules for a source
		r.Get("/sources/{sourceID}/rules", h.ListRules)
		r.Post("/sources/{sourceID}/rules", h.CreateRule)
		r.Put("/rules/{ruleID}", h.UpdateRule)
		r.Delete("/rules/{ruleID}", h.DeleteRule)

		// Event log
		r.Get("/events", h.ListEvents)
	})
}
