package auth

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *JWTManager) {
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(JWTValidate(jwtMgr))
		r.Post("/auth/logout", h.Logout)
		r.Get("/auth/me", h.Me)
		r.Put("/auth/password", h.ChangePassword)
		r.Post("/auth/sse-ticket", h.IssueSSETicket)
	})
}
