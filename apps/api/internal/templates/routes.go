package templates

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/templates", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// All authenticated users can list/get (needed for the new-case form picker)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)

		// Apply also available to all — creates tasks under a case they own
		r.Post("/{id}/apply", h.Apply)

		// Manage templates: admin / super_admin / manager
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin, db.UserRoleManager))
			r.Post("/", h.Create)
			r.Put("/{id}", h.Update)
			r.Delete("/{id}", h.Delete)
		})
	})
}
