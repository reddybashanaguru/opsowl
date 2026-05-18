package auth

import "github.com/go-chi/chi/v5"

func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *JWTManager) {
	r.Post("/auth/login", h.Login)
	r.Post("/auth/refresh", h.Refresh)
	// MFA challenge: accepts partial MFA token (MFAPending=true), no JWTValidate wrapper.
	r.Post("/auth/mfa/challenge", h.MFAChallenge)

	r.Group(func(r chi.Router) {
		r.Use(JWTValidate(jwtMgr))
		r.Post("/auth/logout", h.Logout)
		r.Get("/auth/me", h.Me)
		r.Put("/auth/password", h.ChangePassword)
		r.Post("/auth/sse-ticket", h.IssueSSETicket)
		// MFA management (requires full authenticated session).
		r.Get("/auth/mfa/status", h.MFAStatus)
		r.Post("/auth/mfa/enroll", h.MFAEnroll)
		r.Post("/auth/mfa/verify-enroll", h.MFAVerifyEnroll)
		r.Post("/auth/mfa/disable", h.MFADisable)
	})
}
