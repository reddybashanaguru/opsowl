package findings

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// RegisterRoutes mounts all findings endpoints on the provided router.
func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Read — viewers and above
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/findings", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/findings/summary", h.Summary)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/findings/{findingId}", h.Get)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/findings/{findingId}/evidence", h.ListEvidence)

		// Write — analyst and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/findings", h.Create)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Patch("/cases/{id}/findings/{findingId}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{id}/findings/{findingId}", h.Delete)

		// Status transitions — analyst+ (service enforces maker-checker for approve/reject)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/findings/{findingId}/submit", h.Submit)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/findings/{findingId}/approve", h.Approve)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/findings/{findingId}/reject", h.Reject)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/findings/{findingId}/contest", h.Contest)

		// Evidence management — analyst and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/findings/{findingId}/evidence", h.LinkEvidence)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{id}/findings/{findingId}/evidence/{linkId}", h.RemoveEvidence)
	})
}
