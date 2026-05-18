package config

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	// Public — frontend needs these before login to render labels
	r.Get("/config", h.Get)
	r.Get("/config/defaults", h.Defaults)

	// Admin-only update
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.With(auth.RequireRole("admin", "super_admin")).Put("/config", h.Update)
	})
}
