package users

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) Create(ctx context.Context, params db.CreateUserParams) (db.CreateUserRow, error) {
	return r.q.CreateUser(ctx, params)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *Repository) List(ctx context.Context, limit int32, role db.NullUserRole, cursor uuid.NullUUID) ([]db.ListUsersRow, error) {
	return r.q.ListUsers(ctx, db.ListUsersParams{
		Limit:  limit,
		Role:   role,
		Cursor: cursor,
	})
}

func (r *Repository) Count(ctx context.Context, role db.NullUserRole) (int64, error) {
	return r.q.CountUsers(ctx, role)
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fullName *string, role *db.UserRole, isActive *bool) (db.UpdateUserRow, error) {
	params := db.UpdateUserParams{ID: id}
	if fullName != nil {
		params.FullName = sql.NullString{String: *fullName, Valid: true}
	}
	if role != nil {
		params.Role = db.NullUserRole{UserRole: *role, Valid: true}
	}
	if isActive != nil {
		params.IsActive = sql.NullBool{Bool: *isActive, Valid: true}
	}
	return r.q.UpdateUser(ctx, params)
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) (db.SoftDeleteUserRow, error) {
	return r.q.SoftDeleteUser(ctx, id)
}

func (r *Repository) RevokeAllTokens(ctx context.Context, userID uuid.UUID) error {
	_ = r.q.RevokeAllUserRefreshTokens(ctx, userID)
	_ = r.q.RevokeAllUserSessions(ctx, userID)
	return nil
}

func (r *Repository) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	return r.q.AdminUnlockAccount(ctx, userID)
}
