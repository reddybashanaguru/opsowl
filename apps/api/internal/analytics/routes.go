package analytics

import (
	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *auth.JWTManager) {
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/analytics/workload", h.Workload)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/analytics/workload/export", h.ExportWorkload)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/analytics/risk", h.RiskLeaderboard)
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/analytics/risk/export", h.ExportRisk)
	})
}
