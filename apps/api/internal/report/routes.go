package report

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))
		// PDF case report — any investigator role
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/cases/{id}/report", h.Generate)
		// Regulatory templates — read-only, any investigator role
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/regulatory/templates", h.ListRegulatoryTemplates)
		// STR/SAR/CTR generation — compliance officers and above ONLY.
		// PMLA s.8A (tipping-off): analysts must not have access to confirm
		// whether a subject is under investigation.
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/regulatory-report", h.GenerateRegulatoryReport)
	})
}
