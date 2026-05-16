package users

import (
	"time"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type CreateUserRequest struct {
	Email    string      `json:"email"`
	FullName string      `json:"full_name"`
	Role     db.UserRole `json:"role"`
	Password string      `json:"password"`
}

type UpdateUserRequest struct {
	FullName *string      `json:"full_name,omitempty"`
	Role     *db.UserRole `json:"role,omitempty"`
	IsActive *bool        `json:"is_active,omitempty"`
}

type UserResponse struct {
	ID          uuid.UUID   `json:"id"`
	Email       string      `json:"email"`
	FullName    string      `json:"full_name"`
	Role        db.UserRole `json:"role"`
	IsActive    bool        `json:"is_active"`
	LastLoginAt *string     `json:"last_login_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type ListUsersResponse struct {
	Users      []UserResponse `json:"users"`
	NextCursor *string        `json:"next_cursor"`
	HasMore    bool           `json:"has_more"`
	TotalCount int64          `json:"total_count"`
}
