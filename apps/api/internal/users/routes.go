package users

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// manager+ can list users
		r.With(auth.RequireRole(db.UserRoleManager, db.UserRoleAdmin, db.UserRoleSuperAdmin)).
			Get("/users", h.List)

		// any authenticated user can view a specific user
		r.Get("/users/{id}", h.GetByID)

		// admin+ can create/delete/unlock
		r.With(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin)).
			Post("/users", h.Create)
		r.With(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin)).
			Delete("/users/{id}", h.Delete)
		r.With(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin)).
			Post("/users/{id}/unlock", h.Unlock)

		// self or admin+ can update
		r.Put("/users/{id}", h.Update)
	})
}
