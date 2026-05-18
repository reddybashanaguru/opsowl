package auth

import (
	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
	User         UserSummary `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// MFALoginResponse is returned instead of a full token pair when the user has TOTP enabled.
type MFALoginResponse struct {
	MFARequired bool   `json:"mfa_required"`
	MFAToken    string `json:"mfa_token"` // 5-min partial JWT; pass to /auth/mfa/challenge
}

type MFAChallengeRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"` // 6-digit TOTP code or 8-char backup code
}

type MFAEnrollResponse struct {
	Secret      string   `json:"secret"`       // Base32 TOTP secret (for manual entry)
	OTPAuthURL  string   `json:"otpauth_url"`  // otpauth:// URI for QR code
	BackupCodes []string `json:"backup_codes"` // 8 single-use codes
}

type MFAVerifyRequest struct {
	Code string `json:"code"` // 6-digit TOTP code to confirm enrollment
}

type MFADisableRequest struct {
	Password string `json:"password"`
}

type UserSummary struct {
	ID          uuid.UUID   `json:"id"`
	Email       string      `json:"email"`
	FullName    string      `json:"full_name"`
	Role        db.UserRole `json:"role"`
	LastLoginAt *string     `json:"last_login_at,omitempty"`
}
