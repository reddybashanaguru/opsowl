package narrative

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Read — viewers and above
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/narratives", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/narratives/{narrativeId}", h.Get)

		// Generate and edit — analyst and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/narratives/generate", h.Generate)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Patch("/cases/{id}/narratives/{narrativeId}", h.Edit)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{id}/narratives/{narrativeId}", h.Delete)

		// Approve and reject — manager and above (service enforces 4-eye)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/narratives/{narrativeId}/approve", h.Approve)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/narratives/{narrativeId}/reject", h.Reject)
	})
}
