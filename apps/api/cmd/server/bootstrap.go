package main

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// seedAdminUser creates the first super_admin on fresh deployments.
// Only runs when SEED_ADMIN_EMAIL + SEED_ADMIN_PASSWORD are set AND the
// users table is empty — subsequent boots are always no-ops.
func seedAdminUser(ctx context.Context, db *sql.DB, logger *zap.Logger) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_ADMIN_EMAIL")))
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return
	}
	name := strings.TrimSpace(os.Getenv("SEED_ADMIN_NAME"))
	if name == "" {
		name = "System Administrator"
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&count); err != nil {
		logger.Warn("bootstrap: could not count users", zap.Error(err))
		return
	}
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		logger.Error("bootstrap: failed to hash admin password", zap.Error(err))
		return
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, full_name, role, is_active)
		VALUES ($1, $2, $3, 'super_admin', true)`,
		email, string(hash), name)
	if err != nil {
		logger.Error("bootstrap: failed to seed admin user", zap.Error(err))
		return
	}
	logger.Info("bootstrap: initial super_admin created", zap.String("email", email))
}
