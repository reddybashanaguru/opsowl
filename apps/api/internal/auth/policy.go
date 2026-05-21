package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

var (
	ErrPasswordTooShort    = errors.New("password_too_short")
	ErrPasswordNoUppercase = errors.New("password_no_uppercase")
	ErrPasswordNoNumber    = errors.New("password_no_number")
	ErrPasswordReused      = errors.New("password_recently_used")
	ErrPasswordExpired     = errors.New("password_expired")
)

// PasswordPolicy holds the configurable password rules for the workspace.
type PasswordPolicy struct {
	MinLength      int
	ExpiryDays     int  // 0 = never expires
	HistoryCount   int  // 0 = no history check
	RequireUpper   bool
	RequireNumber  bool
}

// Validate checks that the password meets all policy requirements.
func (p PasswordPolicy) Validate(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("%w: minimum %d characters required", ErrPasswordTooShort, p.MinLength)
	}
	var hasUpper, hasNumber bool
	for _, c := range password {
		if unicode.IsUpper(c) {
			hasUpper = true
		}
		if unicode.IsDigit(c) {
			hasNumber = true
		}
	}
	if p.RequireUpper && !hasUpper {
		return ErrPasswordNoUppercase
	}
	if p.RequireNumber && !hasNumber {
		return ErrPasswordNoNumber
	}
	return nil
}

// CheckHistory verifies the new password has not been used in the last N passwords.
func CheckHistory(ctx context.Context, q *db.Queries, userID uuid.UUID, newPassword string, historyCount int) error {
	if historyCount <= 0 {
		return nil
	}
	recent, err := q.GetPasswordHistory(ctx, db.GetPasswordHistoryParams{
		UserID: userID,
		Limit:  int32(historyCount),
	})
	if err != nil {
		return nil // graceful — don't block on history check failure
	}
	h := sha256.Sum256([]byte(newPassword))
	newHash := fmt.Sprintf("%x", h)
	for _, rec := range recent {
		if rec.PasswordHash == newHash {
			return ErrPasswordReused
		}
	}
	return nil
}

// RecordHistory adds the current password hash to the history table and
// prunes entries beyond historyCount to keep the table lean.
func RecordHistory(ctx context.Context, q *db.Queries, userID uuid.UUID, passwordHash string, historyCount int) {
	h := sha256.Sum256([]byte(passwordHash))
	_ = q.AddPasswordHistory(ctx, db.AddPasswordHistoryParams{
		UserID:       userID,
		PasswordHash: fmt.Sprintf("%x", h),
	})
	if historyCount > 0 {
		_ = q.PrunePasswordHistory(ctx, db.PrunePasswordHistoryParams{
			UserID: userID,
			Limit:  int32(historyCount),
		})
	}
}

// IsPasswordExpired returns true when the user's password has exceeded the policy expiry.
func IsPasswordExpired(user db.User, expiryDays int) bool {
	if expiryDays <= 0 {
		return false
	}
	if user.PasswordChangedAt.Valid {
		return time.Now().After(user.PasswordChangedAt.Time.Add(time.Duration(expiryDays) * 24 * time.Hour))
	}
	// If never changed, treat the account creation date as the baseline.
	return time.Now().After(user.CreatedAt.Add(time.Duration(expiryDays) * 24 * time.Hour))
}

// User model must include PasswordChangedAt — add a nullable helper for the model extension.
type UserPasswordFields struct {
	PasswordChangedAt sql.NullTime
	PasswordExpiresAt sql.NullTime
}
