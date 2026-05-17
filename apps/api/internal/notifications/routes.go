package notifications

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/notifications", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/notifications/unread-count", h.UnreadCount)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Put("/notifications/read-all", h.MarkAllRead)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Put("/notifications/{id}/read", h.MarkRead)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/notifications/preferences", h.GetPreferences)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Put("/notifications/preferences/{type}", h.UpdatePreference)
	})

	// SSE — auth via ?token= query param
	r.Get("/notifications/stream", h.Stream)
}
