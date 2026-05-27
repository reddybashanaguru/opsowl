package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/reddybashanaguru/opsowl/internal/response"
)

// validatePasswordComplexity enforces enterprise password policy:
// min 8 chars, at least one uppercase, one lowercase, one digit, one special char.
func validatePasswordComplexity(pw string) error {
	if len(pw) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range pw {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}

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

	result, err := h.svc.Login(r.Context(), req.Email, req.Password, realIP(r), r.UserAgent())
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			response.Err(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		case errors.Is(err, ErrAccountDisabled):
			response.Err(w, http.StatusForbidden, "ACCOUNT_DISABLED", "this account has been deactivated")
		case errors.Is(err, ErrAccountLocked):
			response.Err(w, http.StatusTooManyRequests, "ACCOUNT_LOCKED", "account temporarily locked due to repeated failed login attempts")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		}
		return
	}

	// MFA step required — return partial response (no cookie).
	if result.MFARequired {
		response.JSON(w, http.StatusOK, MFALoginResponse{
			MFARequired: true,
			MFAToken:    result.MFAToken,
		})
		return
	}

	pair := result.Pair
	http.SetCookie(w, refreshCookie(pair.RefreshToken, int(refreshTTL.Seconds())))
	http.SetCookie(w, accessCookie(pair.AccessToken, defaultAccessTTLSec))

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
	http.SetCookie(w, accessCookie(pair.AccessToken, defaultAccessTTLSec))
	response.JSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	rawToken := tokenFromCookieOrBody(r)
	claims := ClaimsFromCtx(r.Context())

	if claims != nil {
		// Revoke the access token immediately so it cannot be replayed within its TTL.
		h.svc.RevokeCurrentAccessToken(r.Context(), claims)
	}
	if rawToken != "" && claims != nil {
		_ = h.svc.Logout(r.Context(), rawToken, claims.UserID, realIP(r))
	}
	http.SetCookie(w, expireRefreshCookie())
	http.SetCookie(w, expireAccessCookie())
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
	if err := validatePasswordComplexity(req.NewPassword); err != nil {
		response.Err(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
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

// MFAStatus returns whether the authenticated user has MFA active.
func (h *Handler) MFAStatus(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	active, err := h.svc.repo.HasActiveTOTP(r.Context(), claims.UserID)
	if err != nil {
		active = false
	}
	response.JSON(w, http.StatusOK, map[string]bool{"active": active})
}

// MFAChallenge validates the MFA token + TOTP code and returns a full token pair.
func (h *Handler) MFAChallenge(w http.ResponseWriter, r *http.Request) {
	var req MFAChallengeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.MFAToken == "" || req.Code == "" {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "mfa_token and code are required")
		return
	}

	result, err := h.svc.VerifyMFAChallenge(r.Context(), req.MFAToken, req.Code, realIP(r))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMFACode), errors.Is(err, ErrInvalidCredentials):
			response.Err(w, http.StatusUnauthorized, "INVALID_MFA_CODE", "invalid or expired code")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "MFA challenge failed")
		}
		return
	}

	pair := result.Pair
	http.SetCookie(w, refreshCookie(pair.RefreshToken, int(refreshTTL.Seconds())))
	http.SetCookie(w, accessCookie(pair.AccessToken, defaultAccessTTLSec))

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

// MFAEnroll starts TOTP enrollment and returns QR URL + backup codes.
func (h *Handler) MFAEnroll(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	resp, err := h.svc.EnrollMFA(r.Context(), claims.UserID, claims.Email)
	if err != nil {
		if errors.Is(err, ErrMFAAlreadyActive) {
			response.Err(w, http.StatusConflict, "MFA_ALREADY_ACTIVE", "MFA is already enabled — disable it first")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not start MFA enrollment")
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// MFAVerifyEnroll confirms a TOTP code to complete enrollment.
func (h *Handler) MFAVerifyEnroll(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req MFAVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "code is required")
		return
	}

	if err := h.svc.VerifyMFAEnrollment(r.Context(), claims.UserID, req.Code); err != nil {
		switch {
		case errors.Is(err, ErrInvalidMFACode):
			response.Err(w, http.StatusBadRequest, "INVALID_MFA_CODE", "code is incorrect — check your authenticator app")
		case errors.Is(err, ErrMFAEnrollmentPending):
			response.Err(w, http.StatusConflict, "NO_PENDING_ENROLLMENT", "start enrollment first")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not activate MFA")
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "MFA enabled successfully"})
}

// MFADisable disables TOTP after verifying the user's password.
func (h *Handler) MFADisable(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromCtx(r.Context())
	if claims == nil {
		response.Err(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}

	var req MFADisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "password is required")
		return
	}

	if err := h.svc.DisableMFA(r.Context(), claims.UserID, req.Password, realIP(r)); err != nil {
		switch {
		case errors.Is(err, ErrIncorrectPassword):
			response.Err(w, http.StatusBadRequest, "INCORRECT_PASSWORD", "password is incorrect")
		case errors.Is(err, ErrMFANotActive):
			response.Err(w, http.StatusConflict, "MFA_NOT_ACTIVE", "MFA is not enabled")
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not disable MFA")
		}
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "MFA disabled"})
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

// accessCookie returns an httpOnly cookie carrying the short-lived JWT access token.
// Path "/" so every API route receives it automatically; Secure in production.
func accessCookie(token string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureEnv(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

func expireAccessCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureEnv(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
}

const defaultAccessTTLSec = 3600 // 1 hour — mirrors JWTManager default
