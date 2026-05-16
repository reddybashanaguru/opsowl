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

type UserSummary struct {
	ID          uuid.UUID   `json:"id"`
	Email       string      `json:"email"`
	FullName    string      `json:"full_name"`
	Role        db.UserRole `json:"role"`
	LastLoginAt *string     `json:"last_login_at,omitempty"`
}
