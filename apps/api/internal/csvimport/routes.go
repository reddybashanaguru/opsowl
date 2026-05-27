package csvimport

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/imports", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Use(auth.RequireRole(
			db.UserRoleAdmin, db.UserRoleSuperAdmin, db.UserRoleManager,
		))

		r.Get("/columns", h.ListColumns)
		r.Post("/preview", h.Preview)
		r.Get("/", h.ListJobs)
		r.Post("/", h.StartImport)
		r.Get("/{id}", h.GetJob)
		r.Get("/{id}/rows", h.ListRows)
	})
}
