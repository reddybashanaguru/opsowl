package playbooks

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/playbooks", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// All authenticated users can list/view playbooks
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)

		// Only admin/super_admin can create/modify/delete
		r.With(auth.RequireRole("admin", "super_admin")).Post("/", h.Create)
		r.With(auth.RequireRole("admin", "super_admin")).Patch("/{id}", h.Update)
		r.With(auth.RequireRole("admin", "super_admin")).Put("/{id}/steps", h.ReplaceSteps)
		r.With(auth.RequireRole("admin", "super_admin")).Delete("/{id}", h.Delete)
	})
}
