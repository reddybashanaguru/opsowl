package notes

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Notes CRUD (case-scoped)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/notes", h.List)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/notes", h.Create)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/notes/{note_id}", h.Get)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Put("/cases/{id}/notes/{note_id}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{id}/notes/{note_id}", h.Delete)

		// Comments
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/notes/{note_id}/comments", h.ListComments)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/comments", h.CreateComment)

		// Combined activity feed
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/notes-activity", h.ListActivity)

		// Mentions
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/mentions/unread-count", h.UnreadCount)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Put("/mentions/{id}/read", h.MarkMentionRead)
	})

	// SSE — no JWTValidate middleware (auth via ?token= query param)
	r.Get("/cases/{id}/notes/stream", h.Stream)
}
