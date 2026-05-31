package entity_graph

import (
	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, sqlDB *sql.DB, jwtMgr *auth.JWTManager) {
	repo := NewRepository(sqlDB)
	svc := NewService(repo)
	h := NewHandler(svc)

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// All roles from analyst up can view and manage entity relationships
		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/relationships", h.List)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Post("/subjects/{id}/relationships", h.Create)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Delete("/subjects/{id}/relationships/{rel_id}", h.Delete)

		r.With(auth.RequireRole("analyst", "manager", "admin", "super_admin")).
			Get("/subjects/{id}/network", h.GetNetwork)
	})
}
