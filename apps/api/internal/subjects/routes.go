package subjects

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// Subject directory — all authenticated users can read
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects", h.List)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}", h.Get)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/cases", h.GetCases)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/filings", h.GetFilings)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/risk-history", h.GetRiskHistory)

		// Mutations — analyst+ can create/update subjects
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/subjects", h.Create)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Patch("/subjects/{id}", h.Update)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/subjects/{id}/watchlist", h.SetWatchlist)

		// Delete — manager+ only
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Delete("/subjects/{id}", h.Delete)

		// Fuzzy Entity Resolution
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/similar", h.GetSimilar)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/subjects/{id}/confirm-match", h.ConfirmMatch)

		// Case ↔ Subject linking (mounted under /cases/{caseId}/subjects)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/cases/{caseId}/subjects", h.ListForCase)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/cases/{caseId}/subjects", h.LinkToCase)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/cases/{caseId}/subjects/{subjectId}", h.UnlinkFromCase)
	})
}
