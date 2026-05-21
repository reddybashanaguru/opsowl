package documents

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Read routes — API key callers need documents:read scope.
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("documents:read")).
			Get("/documents", h.ListAll)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("documents:read")).
			Get("/cases/{caseId}/documents", h.List)
		r.With(auth.RequireRole("viewer", "analyst", "manager", "admin", "super_admin"),
			auth.RequireScope("documents:read")).
			Get("/cases/{caseId}/documents/{docId}", h.Get)

		// Write routes — not accessible to API keys (no documents:write scope exists).
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/documents", h.Upload)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{caseId}/documents/{docId}", h.Delete)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/documents/{docId}/retry", h.Retry)
	})
}
