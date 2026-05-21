package cases

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// viewer+ — read
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}", h.Get)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/activity", h.ListActivity)

		// analyst+ — create, update, tags
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases", h.Create)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Put("/cases/{id}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/tags", h.UpdateTags)

		// role-gated — status change and assign
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/status", h.ChangeStatus)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/assign", h.Assign)

		// admin+ — delete
		r.With(auth.RequireRole("admin", "super_admin")).
			Delete("/cases/{id}", h.Delete)

		// Bulk operations (manager+)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/bulk/status", h.BulkStatus)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/bulk/assign", h.BulkAssign)
		r.With(auth.RequireRole("admin", "super_admin")).
			Post("/cases/bulk/delete", h.BulkDelete)
	})
}
