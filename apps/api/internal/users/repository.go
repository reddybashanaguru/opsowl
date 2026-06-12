package users

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q  *db.Queries
	db *sql.DB
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// WithDB wires in a raw *sql.DB for queries that cross module boundaries
// (e.g., the DPDP erasure PMLA hold check against str_filings).
func (r *Repository) WithDB(d *sql.DB) *Repository {
	r.db = d
	return r
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

// HardDelete permanently removes the user row — only for test/cleanup use by super_admin.
func (r *Repository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("raw db not configured")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *Repository) RevokeAllTokens(ctx context.Context, userID uuid.UUID) error {
	_ = r.q.RevokeAllUserRefreshTokens(ctx, userID)
	_ = r.q.RevokeAllUserSessions(ctx, userID)
	return nil
}

func (r *Repository) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	return r.q.AdminUnlockAccount(ctx, userID)
}

// HasActiveSTRHold returns true when the user is referenced as submitted_by or
// approved_by in any STR filing created within the last 10 years. Under PMLA s.12
// those records must be retained, so PII erasure is blocked.
func (r *Repository) HasActiveSTRHold(ctx context.Context, userID uuid.UUID) (bool, error) {
	if r.db == nil {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM str_filings
			WHERE (submitted_by = $1 OR approved_by = $1)
			  AND created_at > NOW() - INTERVAL '10 years'
		)`, userID).Scan(&exists)
	return exists, err
}

// ErasePII anonymises all personally identifiable fields for the given user.
// erased_at is stamped; all active sessions are revoked. The user row is
// kept so that foreign-key references in audit_logs / str_filings remain intact.
func (r *Repository) ErasePII(ctx context.Context, userID uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("raw db not configured")
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		   SET email         = 'erased_' || LEFT(id::TEXT, 8) || '@erased.invalid',
		       full_name     = '[ERASED]',
		       password_hash = '',
		       erased_at     = NOW(),
		       updated_at    = NOW()
		 WHERE id = $1 AND erased_at IS NULL`, userID)
	if err != nil {
		return err
	}
	_ = r.q.RevokeAllUserRefreshTokens(ctx, userID)
	_ = r.q.RevokeAllUserSessions(ctx, userID)
	return nil
}
