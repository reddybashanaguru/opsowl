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

		r.Get("/audit", h.List)
		r.Get("/audit/export", h.ExportCSV)
		r.Get("/audit/stats", h.Stats)
	})
}
