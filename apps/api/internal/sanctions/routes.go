package sanctions

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Per-subject screening and hits — analysts+
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/subjects/{id}/sanctions/screen", h.ScreenSubject)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/sanctions/hits", h.GetHits)

		// Hit review — manager+
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Patch("/sanctions/hits/{hitId}", h.AcknowledgeHit)

		// List metadata + global hits dashboard — manager+
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/sanctions/lists", h.GetLists)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/sanctions/hits", h.GetAllHits)
	})
}
