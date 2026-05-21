package cases

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// viewer+ — read; API key callers need cases:read scope
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:read")).
			Get("/cases", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:read")).
			Get("/cases/{id}", h.Get)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:read")).
			Get("/cases/{id}/activity", h.ListActivity)

		// analyst+ — create, update, tags; API key callers need cases:write scope
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases", h.Create)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Put("/cases/{id}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases/{id}/tags", h.UpdateTags)

		// role-gated — status change and assign; need cases:write
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases/{id}/status", h.ChangeStatus)
		r.With(auth.RequireRole("manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases/{id}/assign", h.Assign)

		// admin+ — delete; need cases:write (API keys cannot delete in practice as they're analyst role)
		r.With(auth.RequireRole("admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Delete("/cases/{id}", h.Delete)

		// Bulk operations (manager+); need cases:write
		r.With(auth.RequireRole("manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases/bulk/status", h.BulkStatus)
		r.With(auth.RequireRole("manager", "admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases/bulk/assign", h.BulkAssign)
		r.With(auth.RequireRole("admin", "super_admin"),
			auth.RequireScope("cases:write")).
			Post("/cases/bulk/delete", h.BulkDelete)
	})
}
