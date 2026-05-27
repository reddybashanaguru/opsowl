package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// RevokeChecker stores and checks access-token JTIs in revoked_access_tokens.
// An explicit revocation is only needed for the 1-hour window between logout
// and natural token expiry. The table is kept lean by a daily cleanup job.
type RevokeChecker struct {
	db *sql.DB
}

func NewRevokeChecker(db *sql.DB) *RevokeChecker {
	return &RevokeChecker{db: db}
}

// Revoke marks a JTI as revoked until its natural expiry time.
func (rc *RevokeChecker) Revoke(ctx context.Context, jti uuid.UUID, expiresAt time.Time) error {
	_, err := rc.db.ExecContext(ctx,
		`INSERT INTO revoked_access_tokens (jti, expires_at)
		 VALUES ($1, $2)
		 ON CONFLICT (jti) DO NOTHING`,
		jti, expiresAt,
	)
	return err
}

// IsRevoked returns true when the JTI exists in the revocation table and has
// not yet expired (expired entries are no-ops since the JWT itself is also expired).
func (rc *RevokeChecker) IsRevoked(jti uuid.UUID) bool {
	var revoked bool
	_ = rc.db.QueryRowContext(context.Background(),
		`SELECT EXISTS(
		   SELECT 1 FROM revoked_access_tokens
		    WHERE jti = $1 AND expires_at > NOW()
		 )`,
		jti,
	).Scan(&revoked)
	return revoked
}

// Cleanup deletes entries whose tokens have already expired naturally.
// Call daily — these rows serve no purpose after expiry.
func (rc *RevokeChecker) Cleanup(ctx context.Context) error {
	_, err := rc.db.ExecContext(ctx,
		`DELETE FROM revoked_access_tokens WHERE expires_at < NOW()`,
	)
	return err
}
