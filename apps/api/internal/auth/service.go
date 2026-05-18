package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
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

func (s *Service) Login(ctx context.Context, email, password, ipAddress, userAgent string) (*TokenPair, error) {
	// ── LDAP path ─────────────────────────────────────────────────────────────
	if s.ldap != nil {
		return s.loginWithLDAP(ctx, email, password, ipAddress)
	}

	// ── Local password path ───────────────────────────────────────────────────
	return s.loginLocal(ctx, email, password, ipAddress)
}

func (s *Service) loginLocal(ctx context.Context, email, password, ipAddress string) (*TokenPair, error) {
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

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	go s.repo.CreateSession(context.Background(), user.ID, ipAddress, "") //nolint:errcheck
	go s.repo.CreateAuditLog(context.Background(), "auth.login.success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID, "method": "local"}, "info") //nolint:errcheck
	return pair, nil
}

func (s *Service) loginWithLDAP(ctx context.Context, username, password, ipAddress string) (*TokenPair, error) {
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

	pair, err := s.issueTokenPair(ctx, user, ipAddress)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateUserLastLogin(ctx, user.ID)
	go s.repo.CreateSession(context.Background(), user.ID, ipAddress, "") //nolint:errcheck
	go s.repo.CreateAuditLog(context.Background(), "auth.login.success", &user.ID, ipAddress, "users", &user.ID, map[string]any{"session_id": pair.RTID, "method": "ldap"}, "info") //nolint:errcheck
	return pair, nil
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
