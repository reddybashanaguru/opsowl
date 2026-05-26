package str

import (
	"database/sql"

	"github.com/go-chi/chi/v5"

	"github.com/reddybashanaguru/opsowl/internal/auth"
)

func RegisterRoutes(r chi.Router, sqlDB *sql.DB, jwtMgr *auth.JWTManager) {
	repo := NewRepository(sqlDB)
	h := NewHandler(repo)

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTValidate(jwtMgr))

		// STR/SAR/CTR draft creation and listing — compliance officers only
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/str", h.Create)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/cases/{id}/str", h.List)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Get("/cases/{id}/str/{str_id}", h.Get)

		// Workflow transitions
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/str/{str_id}/submit", h.Submit)
		// Approve/reject — manager+ (maker-checker enforced in handler)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/str/{str_id}/approve", h.Approve)
		r.With(auth.RequireRole("manager", "admin", "super_admin")).
			Post("/cases/{id}/str/{str_id}/reject", h.Reject)

		// Mark as filed + FINnet XML export — admin only (post-approval step)
		r.With(auth.RequireRole("admin", "super_admin")).
			Post("/cases/{id}/str/{str_id}/file", h.MarkFiled)
		r.With(auth.RequireRole("admin", "super_admin")).
			Get("/cases/{id}/str/{str_id}/export", h.ExportFINnet)
	})
}
