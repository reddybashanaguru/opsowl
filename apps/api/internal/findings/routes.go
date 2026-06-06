package findings

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// RegisterRoutes mounts all findings endpoints on the provided router.
// All routes require a valid JWT and are nested under the case resource.
func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Finding CRUD
		r.Get("/cases/{id}/findings", h.List)
		r.Post("/cases/{id}/findings", h.Create)
		r.Get("/cases/{id}/findings/summary", h.Summary)
		r.Get("/cases/{id}/findings/{findingId}", h.Get)
		r.Patch("/cases/{id}/findings/{findingId}", h.Update)
		r.Delete("/cases/{id}/findings/{findingId}", h.Delete)

		// Status transitions
		r.Post("/cases/{id}/findings/{findingId}/submit", h.Submit)
		r.Post("/cases/{id}/findings/{findingId}/approve", h.Approve)
		r.Post("/cases/{id}/findings/{findingId}/reject", h.Reject)
		r.Post("/cases/{id}/findings/{findingId}/contest", h.Contest)

		// Evidence management
		r.Get("/cases/{id}/findings/{findingId}/evidence", h.ListEvidence)
		r.Post("/cases/{id}/findings/{findingId}/evidence", h.LinkEvidence)
		r.Delete("/cases/{id}/findings/{findingId}/evidence/{linkId}", h.RemoveEvidence)
	})
}
