package tasks

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Case-scoped tasks
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/tasks", h.ListByCase)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/tasks", h.Create)

		// My tasks (cross-case)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/tasks", h.ListMine)

		// Individual task operations
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/tasks/{id}", h.Get)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Put("/tasks/{id}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/tasks/{id}/complete", h.Complete)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/tasks/{id}", h.Delete)

		// Comments
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/tasks/{id}/comments", h.ListComments)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/tasks/{id}/comments", h.AddComment)
	})
}
