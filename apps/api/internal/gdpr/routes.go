package gdpr

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// RegisterRoutes mounts GDPR DSAR endpoints under /api/v1/gdpr.
// DPO-level access (manager+) required for all write operations.
func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// All authenticated roles can view requests (audit transparency)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/gdpr/dsar", h.List)

		// Manager+ for write operations and irreversible processing
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/gdpr/dsar", h.Create)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Patch("/gdpr/dsar/{id}", h.UpdateStatus)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/gdpr/dsar/{id}/process", h.Process)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/gdpr/dsar/{id}/records", h.GetAnonymisationRecords)
	})
}
