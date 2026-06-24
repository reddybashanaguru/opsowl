package ubo

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// List and create UBO records for a case — analysts and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/cases/{caseId}/ubo", h.List)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/ubo", h.Create)

		// Update/delete a specific UBO record — managers and above (compliance gatekeepers)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Patch("/cases/{caseId}/ubo/{id}", h.UpdateStatus)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Delete("/cases/{caseId}/ubo/{id}", h.Delete)
	})
}
