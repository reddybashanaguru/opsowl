package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required")
		return
	}

	pair, err := h.svc.Login(r.Context(), req.Email, req.Password, realIP(r), r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.Err(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		case errors.Is(err, ErrAccountDisabled):
			response.Err(w, http.StatusForbidden, "ACCOUNT_DISABLED", "this account has been deactivated")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		}
		return
	}

	http.SetCookie(w, refreshCookie(pair.RefreshToken, int(refreshTTL.Seconds())))

	var lastLogin *string
	if pair.User.LastLoginAt.Valid {
		s := pair.User.LastLoginAt.Time.UTC().Format(time.RFC3339)
		lastLogin = &s
	}
	response.JSON(w, http.StatusOK, LoginResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
		User: UserSummary{
			ID:          pair.User.ID,
			Email:       pair.User.Email,
			FullName:    pair.User.FullName,
			Role:        pair.User.Role,
			LastLoginAt: lastLogin,
		},
	})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	rawToken := tokenFromCookieOrBody(r)
	if rawToken == "" {
		response.Err(w, http.StatusUnauthorized, "MISSING_TOKEN", "refresh token is required")
		return
	}

	pair, err := h.svc.Refresh(r.Context(), rawToken, realIP(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenExpired):
			response.Err(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "refresh token has expired")
		case errors.Is(err, ErrTokenReuse):
			// Clear cookie on reuse attack
			http.SetCookie(w, expireRefreshCookie())
			response.Err(w, http.StatusUnauthorized, "TOKEN_REUSE_DETECTED", "security violation detected — please log in again")
		default:
			response.Err(w, http.StatusUnauthorized, "INVALID_TOKEN", "refresh token is invalid")
		}
		return
	}

	http.SetCookie(w, refreshCookie(pair.RefreshToken, int(refreshTTL.Seconds())))
	response.JSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	rawToken := tokenFromCookieOrBody(r)
	claims := ClaimsFromCtx(r.Context())

	if rawToken != "" && claims != nil {
		_ = h.svc.Logout(r.Context(), rawToken, claims.UserID, realIP(r))
	}
	http.SetCookie(w, expireRefreshCookie())
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	user, err := h.svc.repo.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not load user")
		return
	}
	var lastLogin *string
	if user.LastLoginAt.Valid {
		s := user.LastLoginAt.Time.UTC().Format(time.RFC3339)
		lastLogin = &s
	}
	response.JSON(w, http.StatusOK, UserSummary{
		ID:          user.ID,
		Email:       user.Email,
		FullName:    user.FullName,
		Role:        user.Role,
		LastLoginAt: lastLogin,
	})
}

// IssueSSETicket returns a short-lived (90s) JWT for SSE connections.
// Clients call this endpoint first, then use the ticket as ?token= on SSE endpoints.
// This keeps the full access token out of server access logs.
func (h *Handler) IssueSSETicket(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	ticket, err := h.svc.jwt.SignSSETicket(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not issue SSE ticket")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"expires_in": 90,
	})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if len(req.NewPassword) < 8 {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "new password must be at least 8 characters")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), claims.UserID, req.CurrentPassword, req.NewPassword, realIP(r)); err != nil {
		switch {
		case errors.Is(err, ErrIncorrectPassword):
			response.Err(w, http.StatusBadRequest, "INCORRECT_PASSWORD", "current password is incorrect")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not change password")
		}
		return
	}
	// Expire cookie — all sessions revoked
	http.SetCookie(w, expireRefreshCookie())
	response.JSON(w, http.StatusOK, map[string]string{"message": "password changed — please log in again"})
}

// helpers

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return r.RemoteAddr
}

func tokenFromCookieOrBody(r *http.Request) string {
	if c, err := r.Cookie("refresh_token"); err == nil {
		return c.Value
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.RefreshToken
}

func isSecureEnv() bool {
	return os.Getenv("ENVIRONMENT") == "production"
}

func refreshCookie(token string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   isSecureEnv(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

func expireRefreshCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   isSecureEnv(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}
