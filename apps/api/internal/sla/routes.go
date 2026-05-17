package sla

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/sla/policies", h.List)
		r.With(auth.RequireRole("admin", "super_admin")).
			Put("/sla/policies", h.Upsert)
	})
}
