package license_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/reddybashanaguru/opsowl/internal/license"
)

// ── Test key-pair helpers ─────────────────────────────────────────────────────

// generateTestKeyPair produces a fresh RSA-2048 key pair for use in tests.
// Panics on generation failure (should never happen in tests).
func generateTestKeyPair(t *testing.T) (privateKey *rsa.PrivateKey, publicKeyPEM string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generateTestKeyPair: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return priv, pubPEM
}

// issueLicense signs a Claims struct with the given private key and returns the
// raw JWT string.
func issueLicense(t *testing.T, priv *rsa.PrivateKey, claims license.Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign license: %v", err)
	}
	return signed
}

// ── Unit tests ────────────────────────────────────────────────────────────────

func TestLoad_ValidLicense(t *testing.T) {
	priv, pubPEM := generateTestKeyPair(t)
	expires := time.Now().Add(365 * 24 * time.Hour)

	claims := license.Claims{
		OrgName:  "Acme Bank",
		Seats:    25,
		Features: []string{"insights", "narrative", "sanctions"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "opsowl-licensing",
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := issueLicense(t, priv, claims)

	lic := license.New()
	if err := lic.LoadFromTokenString(tok, pubPEM); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !lic.Loaded() {
		t.Error("Loaded() should return true after successful load")
	}
	if lic.OrgName() != "Acme Bank" {
		t.Errorf("OrgName: got %q want %q", lic.OrgName(), "Acme Bank")
	}
	if !lic.Has("insights") {
		t.Error("Has('insights') should be true")
	}
	if !lic.Has("INSIGHTS") {
		t.Error("Has should be case-insensitive")
	}
	if lic.Has("gdpr") {
		t.Error("Has('gdpr') should be false — not in license")
	}
}

func TestLoad_ExpiredLicense(t *testing.T) {
	priv, pubPEM := generateTestKeyPair(t)

	claims := license.Claims{
		OrgName: "Expired Corp",
		Seats:   10,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)), // yesterday
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)),
		},
	}
	tok := issueLicense(t, priv, claims)

	lic := license.New()
	err := lic.LoadFromTokenString(tok, pubPEM)
	if err == nil {
		t.Fatal("expected error for expired license, got nil")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error should mention expiry, got: %v", err)
	}
}

func TestLoad_WrongSigningKey(t *testing.T) {
	priv, _ := generateTestKeyPair(t)       // key used to sign
	_, wrongPubPEM := generateTestKeyPair(t) // different key used to verify

	claims := license.Claims{
		OrgName: "Tampered",
		Seats:   5,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	tok := issueLicense(t, priv, claims)

	lic := license.New()
	err := lic.LoadFromTokenString(tok, wrongPubPEM)
	if err == nil {
		t.Fatal("expected signature error, got nil")
	}
}

func TestLoad_TamperedPayload(t *testing.T) {
	priv, pubPEM := generateTestKeyPair(t)

	claims := license.Claims{
		OrgName: "Legit Corp",
		Seats:   5,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	tok := issueLicense(t, priv, claims)

	// Tamper with the payload section (middle part of the JWT).
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatal("expected 3-part JWT")
	}
	parts[1] = parts[1] + "tampered"
	tampered := strings.Join(parts, ".")

	lic := license.New()
	if err := lic.LoadFromTokenString(tampered, pubPEM); err == nil {
		t.Fatal("expected error for tampered JWT, got nil")
	}
}

func TestSeatCheck(t *testing.T) {
	priv, pubPEM := generateTestKeyPair(t)

	claims := license.Claims{
		OrgName: "Small Bank",
		Seats:   3,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	tok := issueLicense(t, priv, claims)

	lic := license.New()
	_ = lic.LoadFromTokenString(tok, pubPEM)

	if err := lic.SeatCheck(2); err != nil {
		t.Errorf("SeatCheck(2) with 3 seats: expected nil, got %v", err)
	}
	if err := lic.SeatCheck(3); err == nil {
		t.Error("SeatCheck(3) with 3 seats: expected ErrSeatLimitExceeded, got nil")
	}
	if err := lic.SeatCheck(10); err == nil {
		t.Error("SeatCheck(10) with 3 seats: expected ErrSeatLimitExceeded, got nil")
	}
}

func TestNoLicense_PassThrough(t *testing.T) {
	lic := license.New() // not loaded

	if lic.Loaded() {
		t.Error("freshly constructed License should not be Loaded()")
	}
	if !lic.Has("anything") {
		// When no license is loaded Has should return false — callers decide
		// whether to block or allow. Middleware allows all (dev mode).
		// This test just documents the behaviour.
	}
	if err := lic.SeatCheck(999); err != nil {
		t.Errorf("SeatCheck on unloaded license should return nil, got %v", err)
	}
}

// ── Middleware tests ──────────────────────────────────────────────────────────

func TestMiddleware_ExpiredLicense_Returns402(t *testing.T) {
	// Build a license that already passed verification but is now expired
	// by using an almost-immediately-expiring token.  We can't skip load
	// here so we test the middleware's own time.Now() check by building a
	// mock checker instead.
	checker := &mockChecker{
		loaded:    true,
		expiresAt: time.Now().Add(-1 * time.Hour), // already expired
	}

	mw := license.Middleware(checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402 for expired license, got %d", rr.Code)
	}
}

func TestMiddleware_HealthPassesWhenExpired(t *testing.T) {
	checker := &mockChecker{
		loaded:    true,
		expiresAt: time.Now().Add(-1 * time.Hour),
	}
	mw := license.Middleware(checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("/health should pass even when expired, got %d", rr.Code)
	}
}

func TestMiddleware_ValidLicense_Passes(t *testing.T) {
	checker := &mockChecker{
		loaded:    true,
		expiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	mw := license.Middleware(checker)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid license, got %d", rr.Code)
	}
	if rr.Header().Get("X-License-Expiry") == "" {
		t.Error("X-License-Expiry header should be set")
	}
}

func TestFeatureGate_BlocksUnlicensedFeature(t *testing.T) {
	checker := &mockChecker{
		loaded:    true,
		expiresAt: time.Now().Add(24 * time.Hour),
		features:  map[string]bool{"insights": true},
	}
	mw := license.FeatureGate(checker, "sanctions")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sanctions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402 for unlicensed feature, got %d", rr.Code)
	}
}

func TestFeatureGate_AllowsLicensedFeature(t *testing.T) {
	checker := &mockChecker{
		loaded:    true,
		expiresAt: time.Now().Add(24 * time.Hour),
		features:  map[string]bool{"sanctions": true},
	}
	mw := license.FeatureGate(checker, "sanctions")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sanctions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for licensed feature, got %d", rr.Code)
	}
}

// ── mockChecker ───────────────────────────────────────────────────────────────

type mockChecker struct {
	loaded    bool
	expiresAt time.Time
	orgName   string
	features  map[string]bool
	seats     int
}

func (m *mockChecker) Has(f string) bool        { return m.features[strings.ToLower(f)] }
func (m *mockChecker) OrgName() string           { return m.orgName }
func (m *mockChecker) ExpiresAt() time.Time      { return m.expiresAt }
func (m *mockChecker) Loaded() bool              { return m.loaded }
func (m *mockChecker) SeatCheck(n int) error {
	if m.seats > 0 && n >= m.seats {
		return license.ErrSeatLimitExceeded
	}
	return nil
}
