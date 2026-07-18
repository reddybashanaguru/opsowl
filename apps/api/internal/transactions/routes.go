package transactions

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Account registry — analyst and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/accounts", h.CreateAccount)

		// Read — viewers and above
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/transactions", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/transactions/summary", h.Summary)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/transactions/flow", h.FlowGraph)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/transactions/{txnId}", h.Get)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin")).
			Get("/transactions/fields", h.Fields)

		// Write — analyst and above
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/transactions", h.Create)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/transactions/import", h.Import)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{id}/transactions/link", h.Link)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Patch("/cases/{id}/transactions/{txnId}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{id}/transactions/{txnId}", h.Unlink)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Patch("/cases/{id}/transactions/{txnId}/note", h.UpdateNote)
	})
}
