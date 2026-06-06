package narrative

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		r.Get("/cases/{id}/narratives", h.List)
		r.Post("/cases/{id}/narratives/generate", h.Generate)
		r.Get("/cases/{id}/narratives/{narrativeId}", h.Get)
		r.Patch("/cases/{id}/narratives/{narrativeId}", h.Edit)
		r.Post("/cases/{id}/narratives/{narrativeId}/approve", h.Approve)
		r.Post("/cases/{id}/narratives/{narrativeId}/reject", h.Reject)
		r.Delete("/cases/{id}/narratives/{narrativeId}", h.Delete)
	})
}
