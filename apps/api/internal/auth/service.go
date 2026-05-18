package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

const refreshTTL = 30 * 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrAccountDisabled    = errors.New("account_disabled")
	ErrTokenExpired       = errors.New("token_expired")
	ErrTokenRevoked       = errors.New("token_revoked")
	ErrTokenReuse         = errors.New("token_reuse_detected")
	ErrIncorrectPassword  = errors.New("incorrect_password")
	ErrUserNotFound       = errors.New("user_not_found")
	ErrMFARequired        = errors.New("mfa_required")
	ErrInvalidMFACode     = errors.New("invalid_mfa_code")
	ErrMFAAlreadyActive   = errors.New("mfa_already_active")
	ErrMFANotActive       = errors.New("mfa_not_active")
	ErrMFAEnrollmentPending = errors.New("mfa_enrollment_not_started")
)

type Service struct {
	repo *Repository
	jwt  *JWTManager
	ldap *LDAPAuthenticator
}

func NewService(repo *Repository, jwt *JWTManager) *Service {
	return &Service{repo: repo, jwt: jwt}
}

// WithLDAP enables LDAP/AD authentication as the primary auth path.
func (s *Service) WithLDAP(cfg *LDAPConfig) *Service {
	if cfg.Enabled() {
		s.ldap = NewLDAPAuthenticator(cfg)
	}
	return s
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         db.User
	RTID         uuid.UUID
}

// LoginResult wraps either a full token pair or an MFA challenge step.
type LoginResult struct {
	Pair        *TokenPair // non-nil when login is complete
	MFAToken    string     // non-empty when MFA is required
	MFARequired bool
}

func (s *Service) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*LoginResult, error) {
	// ── LDAP path ─────────────────────────────────────────────────────────────
	if s.ldap != nil {
		return s.loginWithLDAP(ctx, email, password, ipAddress)
	}

	// ── Local password path ───────────────────────────────────────────────────
	return s.loginLocal(ctx, email, password, ipAddress)
}

func (s *Service) loginLocal(ctx context.Context, email, password, ipAddress string) (*LoginResult, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			go s.repo.CreateAuditLog(context.Background(), "auth.login.failed", nil, ipAddress, "", nil, map[string]any{"email": email}, "warn") //nolint:errcheck
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		go s.repo.CreateAuditLog(context.Background(), "auth.login.account_disabled", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "warn") //nolint:errcheck
		return nil, ErrAccountDisabled
	}

	if err := checkPassword(user.PasswordHash, password); err != nil {
		go s.repo.CreateAuditLog(context.Background(), "auth.login.failed", &user.ID, ipAddress, "users", &user.ID, map[string]any{"email": email}, "warn") //nolint:errcheck
		return nil, ErrInvalidCredentials
	}

	// If user has active TOTP, issue a short-lived MFA challenge token instead of full access.
	hasMFA, _ := s.repo.HasActiveTOTP(ctx, user.ID)
	if hasMFA {
		mfaToken, err := s.jwt.SignMFAToken(user.ID, user.Email, user.Role)
		if err != nil {
			return nil, err
		}
		go s.repo.CreateAuditLog(context.Background(), "auth.login.mfa_required", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "info") //nolint:errcheck
		return &LoginResult{MFARequired: true, MFAToken: mfaToken}, nil
	}

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	go s.repo.CreateSession(context.Background(), user.ID, ipAddress, "") //nolint:errcheck
	go s.repo.CreateAuditLog(context.Background(), "auth.login.success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID, "method": "local"}, "info") //nolint:errcheck
	return &LoginResult{Pair: pair}, nil
}

func (s *Service) loginWithLDAP(ctx context.Context, username, password, ipAddress string) (*LoginResult, error) {
	// Verify credentials against LDAP/AD
	resolvedEmail, err := s.ldap.Authenticate(username, password)
	if err != nil {
		go s.repo.CreateAuditLog(context.Background(), "auth.login.failed", nil, ipAddress, "", nil, map[string]any{"username": username, "method": "ldap"}, "warn") //nolint:errcheck
		return nil, ErrInvalidCredentials
	}

	// Look up local user record (for roles) — JIT provision if first login
	user, err := s.repo.GetUserByEmail(ctx, resolvedEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// First LDAP login — provision the user as analyst role by default
			user, err = s.repo.ProvisionLDAPUser(ctx, resolvedEmail, username)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if !user.IsActive {
		go s.repo.CreateAuditLog(context.Background(), "auth.login.account_disabled", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "warn") //nolint:errcheck
		return nil, ErrAccountDisabled
	}

	// LDAP users can also enroll TOTP — check before issuing full pair.
	hasMFA, _ := s.repo.HasActiveTOTP(ctx, user.ID)
	if hasMFA {
		mfaToken, err := s.jwt.SignMFAToken(user.ID, user.Email, user.Role)
		if err != nil {
			return nil, err
		}
		go s.repo.CreateAuditLog(context.Background(), "auth.login.mfa_required", &user.ID, ipAddress, "users", &user.ID, map[string]any{"method": "ldap"}, "info") //nolint:errcheck
		return &LoginResult{MFARequired: true, MFAToken: mfaToken}, nil
	}

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	go s.repo.CreateSession(context.Background(), user.ID, ipAddress, "") //nolint:errcheck
	go s.repo.CreateAuditLog(context.Background(), "auth.login.success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID, "method": "ldap"}, "info") //nolint:errcheck
	return &LoginResult{Pair: pair}, nil
}

func (s *Service) Refresh(ctx context.Context, rawToken, ipAddress string) (*TokenPair, error) {
	hash := hashToken(rawToken)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTokenRevoked
		}
		return nil, err
	}

	// Reuse detection: token already revoked
	if rt.RevokedAt.Valid {
		go s.repo.RevokeAllUserRefreshTokens(context.Background(), rt.UserID)           //nolint:errcheck
		go s.repo.CreateAuditLog(context.Background(), "auth.token.reuse_detected", &rt.UserID, ipAddress, "refresh_tokens", &rt.ID, map[string]any{"rt_hash": hash[:8] + "..."}, "critical") //nolint:errcheck
		return nil, ErrTokenReuse
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}

	// Revoke old token before issuing new pair (rotation)
	if err := s.repo.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, err
	}

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	go s.repo.CreateAuditLog(context.Background(), "auth.token.refresh", &user.ID, ipAddress, "refresh_tokens", &rt.ID, map[string]any{"new_rt_id": pair.RTID}, "info") //nolint:errcheck
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string, userID uuid.UUID, ipAddress string) error {
	hash := hashToken(rawToken)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return nil // idempotent
	}
	_ = s.repo.RevokeRefreshToken(ctx, rt.ID)
	go s.repo.CreateAuditLog(context.Background(), "auth.logout", &userID, ipAddress, "users", &userID, map[string]any{}, "info") //nolint:errcheck
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPwd, newPwd, ipAddress string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := checkPassword(user.PasswordHash, currentPwd); err != nil {
		return ErrIncorrectPassword
	}
	newHash, err := HashPassword(newPwd)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, newHash); err != nil {
		return err
	}
	_ = s.repo.RevokeAllUserRefreshTokens(ctx, userID)
	go s.repo.CreateAuditLog(context.Background(), "auth.password.changed", &userID, ipAddress, "users", &userID, map[string]any{}, "info") //nolint:errcheck
	return nil
}

func (s *Service) VerifyToken(tokenStr string) (*Claims, error) {
	return s.jwt.Verify(tokenStr)
}

// EnrollMFA starts TOTP enrollment: generates a secret, stores it as pending, returns QR + backup codes.
// Backup codes are generated and returned now but only activated when VerifyMFAEnrollment succeeds.
func (s *Service) EnrollMFA(ctx context.Context, userID uuid.UUID, email string) (*MFAEnrollResponse, error) {
	if active, _ := s.repo.HasActiveTOTP(ctx, userID); active {
		return nil, ErrMFAAlreadyActive
	}
	// Clean up any previous pending enrollments.
	_ = s.repo.DeletePendingTOTPSecrets(ctx, userID)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "OpsOwl",
		AccountName: email,
	})
	if err != nil {
		return nil, err
	}

	ts, err := s.repo.CreateTOTPSecret(ctx, userID, key.Secret())
	if err != nil {
		return nil, err
	}

	// Generate 8 backup codes (stored hashed, returned plain-text once).
	backupCodes := make([]string, 8)
	hashes := make([]string, 8)
	for i := range backupCodes {
		code, err := generateBackupCode()
		if err != nil {
			return nil, err
		}
		backupCodes[i] = code
		h := sha256.Sum256([]byte(code))
		hashes[i] = fmt.Sprintf("%x", h)
	}

	if err := s.repo.CreateMFABackupCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}

	_ = ts // secret ID stored in DB; activation happens on VerifyMFAEnrollment
	return &MFAEnrollResponse{
		Secret:      key.Secret(),
		OTPAuthURL:  key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

// VerifyMFAEnrollment confirms a TOTP code against the pending secret and activates it.
func (s *Service) VerifyMFAEnrollment(ctx context.Context, userID uuid.UUID, code string) error {
	pending, err := s.repo.GetPendingTOTPSecret(ctx, userID)
	if err != nil {
		return ErrMFAEnrollmentPending
	}
	if !totp.Validate(code, pending.Secret) {
		return ErrInvalidMFACode
	}
	_, err = s.repo.ActivateTOTPSecret(ctx, pending.ID)
	return err
}

// VerifyMFAChallenge validates an MFA token + TOTP code (or backup code) and returns a full token pair.
func (s *Service) VerifyMFAChallenge(ctx context.Context, mfaToken, code, ipAddress string) (*LoginResult, error) {
	claims, err := s.jwt.Verify(mfaToken)
	if err != nil || !claims.MFAPending {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	ts, err := s.repo.GetActiveTOTPSecret(ctx, user.ID)
	if err != nil {
		return nil, ErrMFANotActive
	}

	// Try TOTP code first.
	if totp.Validate(code, ts.Secret) {
		return s.completeMFALogin(ctx, user, ipAddress)
	}

	// Fall back to backup code (SHA-256 hash lookup).
	h := sha256.Sum256([]byte(code))
	codeHash := fmt.Sprintf("%x", h)
	bc, err := s.repo.GetUnusedBackupCode(ctx, user.ID, codeHash)
	if err != nil {
		go s.repo.CreateAuditLog(context.Background(), "auth.mfa.failed", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "warn") //nolint:errcheck
		return nil, ErrInvalidMFACode
	}
	_ = s.repo.MarkBackupCodeUsed(ctx, bc.ID)
	return s.completeMFALogin(ctx, user, ipAddress)
}

func (s *Service) completeMFALogin(ctx context.Context, user db.User, ipAddress string) (*LoginResult, error) {
	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}
	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	go s.repo.CreateSession(context.Background(), user.ID, ipAddress, "") //nolint:errcheck
	go s.repo.CreateAuditLog(context.Background(), "auth.login.mfa_success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID}, "info") //nolint:errcheck
	return &LoginResult{Pair: pair}, nil
}

// DisableMFA disables TOTP for the user after verifying their password.
func (s *Service) DisableMFA(ctx context.Context, userID uuid.UUID, password, ipAddress string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if err := checkPassword(user.PasswordHash, password); err != nil {
		return ErrIncorrectPassword
	}
	if err := s.repo.RevokeTOTPSecret(ctx, userID); err != nil {
		return err
	}
	_ = s.repo.DeleteAllBackupCodes(ctx, userID)
	go s.repo.CreateAuditLog(context.Background(), "auth.mfa.disabled", &userID, ipAddress, "users", &userID, map[string]any{}, "warn") //nolint:errcheck
	return nil
}

// generateBackupCode produces a random 8-character alphanumeric code.
func generateBackupCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i, v := range b {
		b[i] = chars[int(v)%len(chars)]
	}
	return string(b), nil
}

func (s *Service) issueTokenPair(ctx context.Context, user db.User, _ string) (*TokenPair, error) {
	accessToken, err := s.jwt.Sign(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	rawRT, err := generateRawToken()
	if err != nil {
		return nil, err
	}
	rtHash := hashToken(rawRT)
	rt, err := s.repo.CreateRefreshToken(ctx, user.ID, rtHash, time.Now().Add(refreshTTL))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRT,
		ExpiresIn:    3600,
		User:         user,
		RTID:         rt.ID,
	}, nil
}
