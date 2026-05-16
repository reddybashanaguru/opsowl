package timeline

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// viewer+ read
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/timeline", h.List)

		// analyst+ manual event
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/timeline/events", h.Create)
	})

	// SSE stream — auth handled inside handler via ?token= query param (SSE cannot set headers)
	r.Get("/cases/{id}/timeline/stream", h.Stream)
}
