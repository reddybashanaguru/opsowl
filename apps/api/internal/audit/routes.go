package audit

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Use(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin))

		// API key callers need audit:read scope to access these endpoints.
		r.With(auth.RequireScope("audit:read")).Get("/audit", h.List)
		r.With(auth.RequireScope("audit:read")).Get("/audit/export", h.ExportCSV)
		r.With(auth.RequireScope("audit:read")).Get("/audit/stats", h.Stats)
		r.With(auth.RequireScope("audit:read")).Get("/audit/verify", h.Verify)
	})
}
