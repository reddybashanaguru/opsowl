package closure

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// RegisterRoutes mounts all case closure endpoints on the provided router.
func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Pre-closure checklist (read before creating)
		r.Get("/cases/{id}/closure/checklist", h.Checklist)

		// Active closure record
		r.Get("/cases/{id}/closure", h.Get)
		r.Post("/cases/{id}/closure", h.Create)
		r.Patch("/cases/{id}/closure/{closureId}", h.Update)

		// Workflow transitions
		r.Post("/cases/{id}/closure/{closureId}/submit", h.Submit)
		r.Post("/cases/{id}/closure/{closureId}/approve", h.Approve)
		r.Post("/cases/{id}/closure/{closureId}/reject", h.Reject)

		// Reopen a closed case (voids the approved closure)
		r.Post("/cases/{id}/reopen", h.Reopen)
	})
}
