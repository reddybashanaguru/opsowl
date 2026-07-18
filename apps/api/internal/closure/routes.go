package closure

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// RegisterRoutes mounts all case closure endpoints on the provided router.
func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Read — viewers and above
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/closure/checklist", h.Checklist)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/closure", h.Get)

		// Create and edit — analyst and above (service enforces creator-only)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/closure", h.Create)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Patch("/cases/{id}/closure/{closureId}", h.Update)

		// Submit and revise — analyst and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/closure/{closureId}/submit", h.Submit)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/closure/{closureId}/revise", h.Revise)

		// Approve and reject — manager and above (service enforces 4-eye)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/closure/{closureId}/approve", h.Approve)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/closure/{closureId}/reject", h.Reject)

		// Reopen a closed case — admin and above only
		r.With(auth.RequireRole("admin", "super_admin")).
			Post("/cases/{id}/reopen", h.Reopen)
	})
}
