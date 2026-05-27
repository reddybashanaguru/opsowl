package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

const (
	jwtIssuer   = "opsowl-api"
	jwtAudience = "opsowl-web"
)

type Claims struct {
	UserID     uuid.UUID   `json:"sub"`
	Email      string      `json:"email"`
	Role       db.UserRole `json:"role"`
	JTI        uuid.UUID   `json:"jti_uid"` // unique token ID for revocation
	MFAPending bool        `json:"mfa_pending,omitempty"`
	// Scopes is non-nil only for API key callers.
	Scopes []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	accessTTL  time.Duration
	revoker    *RevokeChecker // nil = revocation disabled (dev / non-browser API keys)
}

func NewJWTManager(privateKeyPath, publicKeyPath string, accessTTL time.Duration) (*JWTManager, error) {
	privBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &JWTManager{
		privateKey: privKey,
		publicKey:  pubKey,
		accessTTL:  accessTTL,
	}, nil
}

// WithRevoker attaches a revocation store so Verify rejects explicitly-logged-out tokens.
func (m *JWTManager) WithRevoker(rc *RevokeChecker) *JWTManager {
	m.revoker = rc
	return m
}

func (m *JWTManager) Sign(userID uuid.UUID, email string, role db.UserRole) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		JTI:    uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// SignMFAToken issues a short-lived (5min) partial token for the MFA challenge step.
func (m *JWTManager) SignMFAToken(userID uuid.UUID, email string, role db.UserRole) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:     userID,
		Email:      email,
		Role:       role,
		JTI:        uuid.New(),
		MFAPending: true,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// SignSSETicket issues a short-lived (90s) token for SSE connections.
func (m *JWTManager) SignSSETicket(userID uuid.UUID, email string, role db.UserRole) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		JTI:    uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(90 * time.Second)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

func (m *JWTManager) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.publicKey, nil
	}, jwt.WithIssuer(jwtIssuer), jwt.WithAudience(jwtAudience), jwt.WithLeeway(30*time.Second))
	if err != nil {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Check explicit revocation — only for full access tokens (not MFA partials or SSE tickets).
	// MFA tokens (5min) and SSE tickets (90s) expire before revocation is meaningful.
	if m.revoker != nil && !claims.MFAPending && claims.JTI != uuid.Nil {
		if m.revoker.IsRevoked(claims.JTI) {
			return nil, errors.New("token revoked")
		}
	}

	return claims, nil
}
