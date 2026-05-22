package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/worker"
)

const refreshTTL = 30 * 24 * time.Hour

const (
	maxFailedAttempts = 5           // lock after N consecutive failures
	baseLockoutMins   = 30          // first lockout: 30 min; doubles each time, capped at 24h
	maxLockoutMins    = 24 * 60     // 24 hours cap
)

var (
	ErrInvalidCredentials   = errors.New("invalid_credentials")
	ErrAccountDisabled      = errors.New("account_disabled")
	ErrAccountLocked        = errors.New("account_locked")
	ErrTokenExpired         = errors.New("token_expired")
	ErrTokenRevoked         = errors.New("token_revoked")
	ErrTokenReuse           = errors.New("token_reuse_detected")
	ErrIncorrectPassword    = errors.New("incorrect_password")
	ErrUserNotFound         = errors.New("user_not_found")
	ErrMFARequired          = errors.New("mfa_required")
	ErrInvalidMFACode       = errors.New("invalid_mfa_code")
	ErrMFAAlreadyActive     = errors.New("mfa_already_active")
	ErrMFANotActive         = errors.New("mfa_not_active")
	ErrMFAEnrollmentPending = errors.New("mfa_enrollment_not_started")
)

type Service struct {
	repo        *Repository
	jwt         *JWTManager
	ldap        *LDAPAuthenticator
	maxSessions int
	pwdPolicy   PasswordPolicy
	// mfaEnrollMu prevents concurrent MFA enrollment for the same user
	// (TOCTOU: delete-pending + generate + store must be atomic per user).
	mfaEnrollMu sync.Map // map[uuid.UUID]*sync.Mutex
}

func NewService(repo *Repository, jwt *JWTManager) *Service {
	return &Service{
		repo:        repo,
		jwt:         jwt,
		maxSessions: 5,
		pwdPolicy:   PasswordPolicy{MinLength: 8},
	}
}

func (s *Service) WithMaxSessions(n int) *Service {
	s.maxSessions = n
	return s
}

func (s *Service) WithPasswordPolicy(p PasswordPolicy) *Service {
	s.pwdPolicy = p
	return s
}


// auditAsync enqueues an audit log write on the shared background worker pool.
func (s *Service) auditAsync(event string, userID *uuid.UUID, ip, resource string, resourceID *uuid.UUID, meta map[string]any, severity string) {
	worker.TrySubmit(func() {
		s.repo.CreateAuditLog(context.Background(), event, userID, ip, resource, resourceID, meta, severity) //nolint:errcheck
	})
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
			s.auditAsync("auth.login.failed", nil, ipAddress, "", nil, map[string]any{"email": email}, "warn") //nolint:errcheck
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		s.auditAsync("auth.login.account_disabled", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "warn") //nolint:errcheck
		return nil, ErrAccountDisabled
	}

	// Check lockout before attempting password verification.
	if user.LockedUntil.Valid && time.Now().Before(user.LockedUntil.Time) {
		s.auditAsync("auth.login.account_locked", &user.ID, ipAddress, "users", &user.ID, map[string]any{"locked_until": user.LockedUntil.Time}, "warn") //nolint:errcheck
		return nil, ErrAccountLocked
	}

	if err := checkPassword(user.PasswordHash, password); err != nil {
		go s.handleFailedAttempt(user, ipAddress)
		return nil, ErrInvalidCredentials
	}

	// If user has active TOTP, issue a short-lived MFA challenge token instead of full access.
	hasMFA, _ := s.repo.HasActiveTOTP(ctx, user.ID)
	if hasMFA {
		mfaToken, err := s.jwt.SignMFAToken(user.ID, user.Email, user.Role)
		if err != nil {
			return nil, err
		}
		s.auditAsync("auth.login.mfa_required", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "info") //nolint:errcheck
		return &LoginResult{MFARequired: true, MFAToken: mfaToken}, nil
	}

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	worker.TrySubmit(func() { s.repo.ResetLoginAttempts(context.Background(), user.ID) }) //nolint:errcheck
	go func() {
		bg := context.Background()
		_ = s.repo.EnforceSessionLimit(bg, user.ID, s.maxSessions)
		_ = s.repo.CreateSession(bg, user.ID, ipAddress, "")
	}()
	s.auditAsync("auth.login.success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID, "method": "local"}, "info") //nolint:errcheck
	return &LoginResult{Pair: pair}, nil
}

// handleFailedAttempt increments the failure counter and locks the account
// once maxFailedAttempts is reached. Lockout duration doubles with each lockout event.
func (s *Service) handleFailedAttempt(user db.User, ipAddress string) {
	ctx := context.Background()
	newCount := int(user.FailedLoginAttempts) + 1
	_ = s.repo.IncrementFailedLoginAttempts(ctx, user.ID)
	s.auditAsync("auth.login.failed", &user.ID, ipAddress, "users", &user.ID,
		map[string]any{"attempt": newCount, "max": maxFailedAttempts}, "warn") //nolint:errcheck

	if newCount >= maxFailedAttempts {
		// Progressive lockout: 30 min → 60 min → 120 min → … capped at 24h
		lockoutMins := baseLockoutMins * (1 << int(user.LockoutCount))
		if lockoutMins > maxLockoutMins {
			lockoutMins = maxLockoutMins
		}
		until := time.Now().Add(time.Duration(lockoutMins) * time.Minute)
		_ = s.repo.LockAccount(ctx, user.ID, until)
		s.auditAsync("auth.login.account_locked", &user.ID, ipAddress, "users", &user.ID,
			map[string]any{"locked_until": until, "lockout_minutes": lockoutMins}, "critical") //nolint:errcheck
	}
}

func (s *Service) loginWithLDAP(ctx context.Context, username, password, ipAddress string) (*LoginResult, error) {
	// Verify credentials against LDAP/AD
	resolvedEmail, err := s.ldap.Authenticate(username, password)
	if err != nil {
		s.auditAsync("auth.login.failed", nil, ipAddress, "", nil, map[string]any{"username": username, "method": "ldap"}, "warn") //nolint:errcheck
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
		s.auditAsync("auth.login.account_disabled", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "warn") //nolint:errcheck
		return nil, ErrAccountDisabled
	}

	// LDAP users can also enroll TOTP — check before issuing full pair.
	hasMFA, _ := s.repo.HasActiveTOTP(ctx, user.ID)
	if hasMFA {
		mfaToken, err := s.jwt.SignMFAToken(user.ID, user.Email, user.Role)
		if err != nil {
			return nil, err
		}
		s.auditAsync("auth.login.mfa_required", &user.ID, ipAddress, "users", &user.ID, map[string]any{"method": "ldap"}, "info") //nolint:errcheck
		return &LoginResult{MFARequired: true, MFAToken: mfaToken}, nil
	}

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	go func() {
		bg := context.Background()
		_ = s.repo.EnforceSessionLimit(bg, user.ID, s.maxSessions)
		_ = s.repo.CreateSession(bg, user.ID, ipAddress, "")
	}()
	s.auditAsync("auth.login.success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID, "method": "ldap"}, "info") //nolint:errcheck
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
		worker.TrySubmit(func() { s.repo.RevokeAllUserRefreshTokens(context.Background(), rt.UserID) }) //nolint:errcheck
		s.auditAsync("auth.token.reuse_detected", &rt.UserID, ipAddress, "refresh_tokens", &rt.ID, map[string]any{"rt_hash": hash[:8] + "..."}, "critical") //nolint:errcheck
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

	s.auditAsync("auth.token.refresh", &user.ID, ipAddress, "refresh_tokens", &rt.ID, map[string]any{"new_rt_id": pair.RTID}, "info") //nolint:errcheck
	return pair, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string, userID uuid.UUID, ipAddress string) error {
	hash := hashToken(rawToken)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		return nil // idempotent
	}
	_ = s.repo.RevokeRefreshToken(ctx, rt.ID)
	s.auditAsync("auth.logout", &userID, ipAddress, "users", &userID, map[string]any{}, "info") //nolint:errcheck
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
	// Validate new password against policy
	if err := s.pwdPolicy.Validate(newPwd); err != nil {
		return err
	}
	// Check password history
	if err := CheckHistory(ctx, s.repo.q, userID, newPwd, s.pwdPolicy.HistoryCount); err != nil {
		return err
	}
	newHash, err := HashPassword(newPwd)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, newHash); err != nil {
		return err
	}
	// Record history and update expiry synchronously — if the server crashes here the
	// password was changed but history/expiry would be inconsistent (AI-03 fix).
	RecordHistory(ctx, s.repo.q, userID, newHash, s.pwdPolicy.HistoryCount)
	_ = s.repo.q.UpdatePasswordChangedAt(ctx, db.UpdatePasswordChangedAtParams{
		ID:         userID,
		ExpiryDays: int32(s.pwdPolicy.ExpiryDays),
	})
	_ = s.repo.RevokeAllUserRefreshTokens(ctx, userID)
	// Audit log is synchronous so forensic timeline is complete even on subsequent crash.
	_ = s.repo.CreateAuditLog(ctx, "auth.password.changed", &userID, ipAddress, "users", &userID, map[string]any{}, "info")
	return nil
}

func (s *Service) VerifyToken(tokenStr string) (*Claims, error) {
	return s.jwt.Verify(tokenStr)
}

// enrollMutex returns the per-user mutex for MFA enrollment (lazy-init via sync.Map).
func (s *Service) enrollMutex(userID uuid.UUID) *sync.Mutex {
	mu, _ := s.mfaEnrollMu.LoadOrStore(userID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// EnrollMFA starts TOTP enrollment: generates a secret, stores it as pending, returns QR + backup codes.
// Backup codes are generated and returned now but only activated when VerifyMFAEnrollment succeeds.
// A per-user mutex prevents concurrent enrollments from overwriting each other's secret.
func (s *Service) EnrollMFA(ctx context.Context, userID uuid.UUID, email string) (*MFAEnrollResponse, error) {
	mu := s.enrollMutex(userID)
	mu.Lock()
	defer mu.Unlock()

	if active, _ := s.repo.HasActiveTOTP(ctx, userID); active {
		return nil, ErrMFAAlreadyActive
	}
	// Clean up any previous pending enrollments (safe — we hold the per-user lock).
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
		s.auditAsync("auth.mfa.failed", &user.ID, ipAddress, "users", &user.ID, map[string]any{}, "warn") //nolint:errcheck
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
	go func() {
		bg := context.Background()
		_ = s.repo.EnforceSessionLimit(bg, user.ID, s.maxSessions)
		_ = s.repo.CreateSession(bg, user.ID, ipAddress, "")
	}()
	s.auditAsync("auth.login.mfa_success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID}, "info") //nolint:errcheck
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
	s.auditAsync("auth.mfa.disabled", &userID, ipAddress, "users", &userID, map[string]any{}, "warn") //nolint:errcheck
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
