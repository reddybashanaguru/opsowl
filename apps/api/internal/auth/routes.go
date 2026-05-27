package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts all auth endpoints.
// authLimiter (optional) is a tight per-IP rate-limiter applied to the login,
// refresh, and MFA challenge endpoints to prevent credential-stuffing attacks.
func RegisterRoutes(r chi.Router, h *Handler, jwtMgr *JWTManager, authLimiter ...func(http.Handler) http.Handler) {
	var lim func(http.Handler) http.Handler
	if len(authLimiter) > 0 && authLimiter[0] != nil {
		lim = authLimiter[0]
	}

	withLimiter := func(next http.HandlerFunc) http.Handler {
		if lim != nil {
			return lim(next)
		}
		return next
	}

	r.Method(http.MethodPost, "/auth/login", withLimiter(h.Login))
	r.Method(http.MethodPost, "/auth/refresh", withLimiter(h.Refresh))
	r.Method(http.MethodPost, "/auth/mfa/challenge", withLimiter(h.MFAChallenge))

	r.Group(func(r chi.Router) {
		r.Use(JWTValidate(jwtMgr))
		r.Post("/auth/logout", h.Logout)
		r.Get("/auth/me", h.Me)
		r.Put("/auth/password", h.ChangePassword)
		r.Post("/auth/sse-ticket", h.IssueSSETicket)
		r.Get("/auth/mfa/status", h.MFAStatus)
		r.Post("/auth/mfa/enroll", h.MFAEnroll)
		r.Post("/auth/mfa/verify-enroll", h.MFAVerifyEnroll)
		r.Post("/auth/mfa/disable", h.MFADisable)
	})
}
