// Package license provides JWT-based offline license verification for
// OpsOwl on-premise deployments. A license file is a signed RS256 JWT
// whose claims declare the customer's organisation name, seat limit,
// expiry date, and feature flags. The signing private key never leaves
// OpsOwl; only the matching public key is embedded in this binary.
//
// SOLID adherence:
//   - Single Responsibility: this file only parses and validates license JWTs.
//   - Open/Closed: feature checks go through Has(); adding a new feature name
//     requires no change here.
//   - Liskov / Interface Segregation: callers depend on the Checker interface,
//     not the concrete License struct.
//   - Dependency Inversion: the server depends on license.Checker; this package
//     has no knowledge of HTTP, config, or other domains.
package license

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrNoLicense is returned when no license file path is configured.
var ErrNoLicense = errors.New("no license file configured (LICENSE_FILE env var)")

// ErrExpired is returned when the license has passed its expiry date.
var ErrExpired = errors.New("license has expired")

// ErrInvalidSignature is returned when the JWT signature does not match the
// embedded OpsOwl public key.
var ErrInvalidSignature = errors.New("license signature is invalid")

// ErrSeatLimitExceeded is returned by SeatCheck when active users exceed the
// licensed seat count.
var ErrSeatLimitExceeded = errors.New("licensed seat limit exceeded")

// opsowlPublicKeyPEM is the RSA-2048 public key used to verify all license
// JWTs.  The matching private key is held exclusively by OpsOwl and is used
// only at license issuance time.
//
// In tests, override this variable with a test key pair via LoadFromTokenString.
var opsowlPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2a7bqV7B6nOQ1W6v4Z8y
xLgRkJmPw3yKK+VLJxBfM/UvU/s6YLKoB8M7Qw+VWgU5VgG3xCJYd9PqR4B4fQ
9DjK3kQ3sPvX8BJ5W0zL6qW3VN9CmY8D6j4RCwFz7XB3Tw8RnJ2lKgV0mP5Xs0Z
THIS_IS_A_PLACEHOLDER_KEY_REPLACE_BEFORE_PRODUCTION_USE_7Xs0ZzVq5yRk
5LMnHkQpY2XKz8vL3aS6WoN4zQA7mK2P5cJ8rBfV3tY9DxU1CwHn6eO4gP7jR0l
bWsKtQ2fXa8UPzN4mV3hL5eY6cJ9rKwB7nT0dM2pXvO8aL4qY5hZ3cF6gW1iRJN
HwIDAQAB
-----END PUBLIC KEY-----`

// Claims holds the registered and private claims embedded in a license JWT.
type Claims struct {
	OrgName   string    `json:"org"`
	Seats     int       `json:"seats"`
	Features  []string  `json:"features"`
	ExpiresAt time.Time `json:"-"` // parsed from jwt.RegisteredClaims
	IssuedAt  time.Time `json:"-"`
	jwt.RegisteredClaims
}

// License is the parsed, verified representation of a license JWT.
// Use Checker to interact with it; the struct is exported only for testing.
type License struct {
	mu       sync.RWMutex
	claims   Claims
	features map[string]struct{}
	loaded   bool
}

// Checker is the interface callers depend on.  Mock it in tests.
//
//go:generate mockgen -destination=mock_checker.go -package=license . Checker
type Checker interface {
	// Has reports whether the named feature is included in this license.
	Has(feature string) bool
	// SeatCheck returns ErrSeatLimitExceeded when activeUsers >= licensed seats.
	SeatCheck(activeUsers int) error
	// OrgName returns the customer name embedded in the license.
	OrgName() string
	// ExpiresAt returns the license expiry time (UTC).
	ExpiresAt() time.Time
	// Loaded reports whether a valid license has been loaded.
	Loaded() bool
}

// Ensure *License satisfies Checker at compile time.
var _ Checker = (*License)(nil)

// New returns an empty License.  Call Load or LoadFromTokenString before use.
func New() *License { return &License{} }

// Load reads a license JWT from filePath and verifies it.
func (l *License) Load(filePath string) error {
	if filePath == "" {
		return ErrNoLicense
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read license file: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	return l.LoadFromTokenString(token, opsowlPublicKeyPEM)
}

// LoadFromTokenString parses and validates a raw JWT string against the
// provided PEM-encoded RSA public key.  Pass a test key in unit tests.
func (l *License) LoadFromTokenString(tokenStr, publicKeyPEM string) error {
	pubKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return ErrInvalidSignature
		}
		return fmt.Errorf("%w: %s", ErrInvalidSignature, err.Error())
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return ErrInvalidSignature
	}

	// Resolve expiry from the standard registered claim.
	if claims.ExpiresAt == (time.Time{}) && claims.RegisteredClaims.ExpiresAt != nil {
		claims.ExpiresAt = claims.RegisteredClaims.ExpiresAt.Time
	}
	if claims.IssuedAt == (time.Time{}) && claims.RegisteredClaims.IssuedAt != nil {
		claims.IssuedAt = claims.RegisteredClaims.IssuedAt.Time
	}

	if time.Now().After(claims.ExpiresAt) {
		return fmt.Errorf("%w (expired %s)", ErrExpired, claims.ExpiresAt.Format("2006-01-02"))
	}

	fm := make(map[string]struct{}, len(claims.Features))
	for _, f := range claims.Features {
		fm[strings.ToLower(f)] = struct{}{}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.claims = *claims
	l.features = fm
	l.loaded = true
	return nil
}

// ── Checker interface implementation ─────────────────────────────────────────

func (l *License) Has(feature string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if !l.loaded {
		return false
	}
	_, ok := l.features[strings.ToLower(feature)]
	return ok
}

func (l *License) SeatCheck(activeUsers int) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if !l.loaded || l.claims.Seats <= 0 {
		return nil // unlicensed or unlimited
	}
	if activeUsers >= l.claims.Seats {
		return fmt.Errorf("%w: %d licensed, %d active", ErrSeatLimitExceeded, l.claims.Seats, activeUsers)
	}
	return nil
}

func (l *License) OrgName() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.claims.OrgName
}

func (l *License) ExpiresAt() time.Time {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.claims.ExpiresAt
}

func (l *License) Loaded() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.loaded
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parsePublicKey(pem string) (*rsa.PublicKey, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
	if err != nil {
		return nil, err
	}
	return key, nil
}
