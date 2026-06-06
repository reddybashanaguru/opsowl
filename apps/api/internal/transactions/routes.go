package transactions

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Account registry
		r.Post("/accounts", h.CreateAccount)

		// Case-scoped transaction routes
		r.Get("/cases/{id}/transactions/summary", h.Summary)
		r.Get("/cases/{id}/transactions/flow", h.FlowGraph)
		r.Post("/cases/{id}/transactions/import", h.Import)
		r.Post("/cases/{id}/transactions/link", h.Link)
		r.Get("/cases/{id}/transactions", h.List)
		r.Post("/cases/{id}/transactions", h.Create)
		r.Get("/cases/{id}/transactions/{txnId}", h.Get)
		r.Patch("/cases/{id}/transactions/{txnId}", h.Update)
		r.Delete("/cases/{id}/transactions/{txnId}", h.Unlink)
		r.Patch("/cases/{id}/transactions/{txnId}/note", h.UpdateNote)

		// CSV import field list (no case scope)
		r.Get("/transactions/fields", h.Fields)
	})
}
