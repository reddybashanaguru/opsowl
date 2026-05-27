package webhooks

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Route("/webhooks", func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		r.Use(auth.RequireRole(db.UserRoleAdmin, db.UserRoleSuperAdmin))

		r.Get("/event-types", h.ListEventTypes)
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/deliveries", h.ListDeliveries)
		r.Post("/{id}/deliveries/{deliveryId}/redeliver", h.Redeliver)
	})
}
