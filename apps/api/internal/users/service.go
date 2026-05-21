package users

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrEmailTaken   = errors.New("email_taken")
	ErrUserNotFound = errors.New("user_not_found")
	ErrForbidden    = errors.New("forbidden")
	ErrWeakPassword = errors.New("weak_password")
)

type Service struct {
	repo    *Repository
	authSvc *auth.Service
}

func NewService(repo *Repository, authSvc *auth.Service) *Service {
	return &Service{repo: repo, authSvc: authSvc}
}

func (s *Service) Create(ctx context.Context, actor *auth.Claims, req CreateUserRequest) (UserResponse, error) {
	if err := validatePassword(req.Password); err != nil {
		return UserResponse{}, ErrWeakPassword
	}
	if !isValidRole(req.Role) {
		return UserResponse{}, errors.New("invalid_role")
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return UserResponse{}, err
	}

	row, err := s.repo.Create(ctx, db.CreateUserParams{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		FullName:     strings.TrimSpace(req.FullName),
		Role:         req.Role,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return UserResponse{}, ErrEmailTaken
		}
		return UserResponse{}, err
	}

	return toResponse(db.User{
		ID:       row.ID,
		Email:    row.Email,
		FullName: row.FullName,
		Role:     row.Role,
		IsActive: row.IsActive,
	}), nil
}

func (s *Service) List(ctx context.Context, roleFilter *db.UserRole, cursor *uuid.UUID, limit int32) (ListUsersResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	// fetch one extra to detect has_more
	rows, err := s.repo.List(ctx, limit+1, nullRole(roleFilter), nullUUID(cursor))
	if err != nil {
		return ListUsersResponse{}, err
	}
	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	total, _ := s.repo.Count(ctx, nullRole(roleFilter))

	users := make([]UserResponse, len(rows))
	for i, r := range rows {
		users[i] = toResponse(db.User{
			ID:          r.ID,
			Email:       r.Email,
			FullName:    r.FullName,
			Role:        r.Role,
			IsActive:    r.IsActive,
			LastLoginAt: r.LastLoginAt,
			CreatedAt:   r.CreatedAt,
		})
	}

	var nextCursor *string
	if hasMore && len(users) > 0 {
		s := users[len(users)-1].ID.String()
		nextCursor = &s
	}

	return ListUsersResponse{
		Users:      users,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		TotalCount: total,
	}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (UserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return UserResponse{}, ErrUserNotFound
	}
	return toResponse(user), nil
}

func (s *Service) Update(ctx context.Context, actor *auth.Claims, targetID uuid.UUID, req UpdateUserRequest) (UserResponse, error) {
	// Self-update: analysts can only update their own full_name
	if actor.UserID != targetID {
		if actor.Role == db.UserRoleAnalyst || actor.Role == db.UserRoleViewer {
			return UserResponse{}, ErrForbidden
		}
	}
	// Only admin+ can change roles
	if req.Role != nil && actor.Role != db.UserRoleAdmin && actor.Role != db.UserRoleSuperAdmin {
		return UserResponse{}, ErrForbidden
	}

	row, err := s.repo.Update(ctx, targetID, req.FullName, req.Role, req.IsActive)
	if err != nil {
		return UserResponse{}, ErrUserNotFound
	}
	// Invalidate all active sessions when role or active status changes so the
	// new permissions take effect immediately rather than after token expiry.
	if req.Role != nil || req.IsActive != nil {
		_ = s.repo.RevokeAllTokens(ctx, targetID)
	}
	return toResponse(db.User{
		ID:          row.ID,
		Email:       row.Email,
		FullName:    row.FullName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		LastLoginAt: row.LastLoginAt,
		CreatedAt:   row.CreatedAt,
	}), nil
}

func (s *Service) Delete(ctx context.Context, actor *auth.Claims, targetID uuid.UUID) error {
	if actor.UserID == targetID {
		return errors.New("cannot_delete_self")
	}
	_, err := s.repo.SoftDelete(ctx, targetID)
	if err != nil {
		return ErrUserNotFound
	}
	// Revoke all active sessions for deleted user
	_ = s.repo.RevokeAllTokens(ctx, targetID)
	return nil
}

func (s *Service) Unlock(ctx context.Context, targetID uuid.UUID) error {
	return s.repo.UnlockAccount(ctx, targetID)
}

// helpers

func toResponse(u db.User) UserResponse {
	r := UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
	if u.LastLoginAt.Valid {
		s := u.LastLoginAt.Time.UTC().Format("2006-01-02T15:04:05Z")
		r.LastLoginAt = &s
	}
	return r
}

func nullRole(r *db.UserRole) db.NullUserRole {
	if r == nil {
		return db.NullUserRole{}
	}
	return db.NullUserRole{UserRole: *r, Valid: true}
}

func nullUUID(u *uuid.UUID) uuid.NullUUID {
	if u == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *u, Valid: true}
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return ErrWeakPassword
	}
	var hasUpper, hasDigit bool
	for _, c := range p {
		if unicode.IsUpper(c) {
			hasUpper = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	if !hasUpper || !hasDigit {
		return ErrWeakPassword
	}
	return nil
}

func isValidRole(r db.UserRole) bool {
	switch r {
	case db.UserRoleSuperAdmin, db.UserRoleAdmin, db.UserRoleManager, db.UserRoleAnalyst, db.UserRoleViewer:
		return true
	}
	return false
}
