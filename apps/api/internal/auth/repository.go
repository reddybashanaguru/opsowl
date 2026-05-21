package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/reddybashanaguru/opsowl/internal/auditchain"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q     *db.Queries
	chain *auditchain.ChainWriter
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// WithChain attaches a ChainWriter so auth audit logs participate in the
// tamper-evident hash chain. Call this after the chain writer is initialised.
func (r *Repository) WithChain(cw *auditchain.ChainWriter) *Repository {
	r.chain = cw
	return r
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *Repository) UpdateUserLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.q.UpdateUserLastLogin(ctx, id)
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, userID uuid.UUID, newHash string) error {
	return r.q.UpdateUserPasswordHash(ctx, db.UpdateUserPasswordHashParams{
		ID:           userID,
		PasswordHash: newHash,
	})
}

func (r *Repository) CreateSession(ctx context.Context, userID uuid.UUID, ipStr, userAgent string) error {
	host, _, err := net.SplitHostPort(ipStr)
	if err != nil {
		host = ipStr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ip = net.ParseIP("127.0.0.1")
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	inet := pqtype.Inet{IPNet: net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}, Valid: true}
	_, err = r.q.CreateSession(ctx, db.CreateSessionParams{
		UserID:    userID,
		IpAddress: inet,
		UserAgent: userAgent,
	})
	return err
}

// EnforceSessionLimit revokes sessions so that at most (maxSessions-1) remain active,
// leaving one slot for the new session being created. Uses a single atomic UPDATE
// to avoid the COUNT-then-UPDATE race condition.
func (r *Repository) EnforceSessionLimit(ctx context.Context, userID uuid.UUID, maxSessions int) error {
	if maxSessions <= 0 {
		return nil
	}
	keepNewest := int64(maxSessions) - 1 // reserve one slot for the incoming session
	if keepNewest < 0 {
		keepNewest = 0
	}
	return r.q.EnforceSessionCap(ctx, db.EnforceSessionCapParams{
		UserID:     userID,
		KeepNewest: keepNewest,
	})
}

func (r *Repository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	return r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (db.RefreshToken, error) {
	return r.q.GetRefreshTokenByHash(ctx, hash)
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	return r.q.RevokeRefreshToken(ctx, id)
}

func (r *Repository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return r.q.RevokeAllUserRefreshTokens(ctx, userID)
}

// ProvisionLDAPUser creates a user record on first successful LDAP login.
// Role defaults to "analyst" — an admin must promote them if needed.
func (r *Repository) ProvisionLDAPUser(ctx context.Context, email, fullName string) (db.User, error) {
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		FullName:     fullName,
		PasswordHash: "$ldap$provisioned$no-local-login",
		Role:         "analyst",
	})
	if err != nil {
		return db.User{}, err
	}
	// Map CreateUserRow → User (same fields, different generated type)
	return db.User{
		ID:          row.ID,
		Email:       row.Email,
		FullName:    row.FullName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		LastLoginAt: row.LastLoginAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		DeletedAt:   row.DeletedAt,
	}, nil
}

func (r *Repository) HasActiveTOTP(ctx context.Context, userID uuid.UUID) (bool, error) {
	return r.q.HasActiveTOTP(ctx, userID)
}

func (r *Repository) GetActiveTOTPSecret(ctx context.Context, userID uuid.UUID) (db.TotpSecret, error) {
	return r.q.GetActiveTOTPSecret(ctx, userID)
}

func (r *Repository) GetPendingTOTPSecret(ctx context.Context, userID uuid.UUID) (db.TotpSecret, error) {
	return r.q.GetPendingTOTPSecret(ctx, userID)
}

func (r *Repository) CreateTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) (db.TotpSecret, error) {
	return r.q.CreateTOTPSecret(ctx, db.CreateTOTPSecretParams{UserID: userID, Secret: secret})
}

func (r *Repository) ActivateTOTPSecret(ctx context.Context, id uuid.UUID) (db.TotpSecret, error) {
	return r.q.ActivateTOTPSecret(ctx, id)
}

func (r *Repository) RevokeTOTPSecret(ctx context.Context, userID uuid.UUID) error {
	return r.q.RevokeTOTPSecret(ctx, userID)
}

func (r *Repository) DeletePendingTOTPSecrets(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeletePendingTOTPSecrets(ctx, userID)
}

func (r *Repository) CreateMFABackupCodes(ctx context.Context, userID uuid.UUID, codeHashes []string) error {
	params := make([]db.CreateMFABackupCodesParams, len(codeHashes))
	for i, h := range codeHashes {
		params[i] = db.CreateMFABackupCodesParams{UserID: userID, CodeHash: h}
	}
	_, err := r.q.CreateMFABackupCodes(ctx, params)
	return err
}

func (r *Repository) GetUnusedBackupCode(ctx context.Context, userID uuid.UUID, codeHash string) (db.MfaBackupCode, error) {
	return r.q.GetUnusedBackupCode(ctx, db.GetUnusedBackupCodeParams{UserID: userID, CodeHash: codeHash})
}

func (r *Repository) MarkBackupCodeUsed(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkBackupCodeUsed(ctx, id)
}

func (r *Repository) DeleteAllBackupCodes(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteAllBackupCodes(ctx, userID)
}

func (r *Repository) CreateAuditLog(ctx context.Context, eventCode string, actorID *uuid.UUID, ipAddress, resourceType string, resourceID *uuid.UUID, metadata map[string]any, severity string) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	var actor uuid.NullUUID
	if actorID != nil {
		actor = uuid.NullUUID{UUID: *actorID, Valid: true}
	}
	var resID uuid.NullUUID
	if resourceID != nil {
		resID = uuid.NullUUID{UUID: *resourceID, Valid: true}
	}
	p := db.CreateAuditLogParams{
		EventCode:    eventCode,
		ActorID:      actor,
		IpAddress:    sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		ResourceType: sql.NullString{String: resourceType, Valid: resourceType != ""},
		ResourceID:   resID,
		Metadata:     metaJSON,
		Severity:     severity,
	}
	if r.chain != nil {
		return r.chain.Write(ctx, p)
	}
	return r.q.CreateAuditLog(ctx, p)
}

func (r *Repository) IncrementFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	return r.q.IncrementFailedLoginAttempts(ctx, userID)
}

func (r *Repository) LockAccount(ctx context.Context, userID uuid.UUID, until time.Time) error {
	return r.q.LockAccount(ctx, db.LockAccountParams{ID: userID, LockedUntil: until})
}

func (r *Repository) ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	return r.q.ResetLoginAttempts(ctx, userID)
}

func (r *Repository) AdminUnlockAccount(ctx context.Context, userID uuid.UUID) error {
	return r.q.AdminUnlockAccount(ctx, userID)
}
