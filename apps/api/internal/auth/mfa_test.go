package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// ── Service-layer MFA tests ───────────────────────────────────────────────────

func TestServiceMFAEnroll_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "enroll@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "enroll@mfa.com", "TestP@ss1")

	resp, err := svc.EnrollMFA(context.Background(), pair.User.ID, "enroll@mfa.com")
	if err != nil {
		t.Fatalf("EnrollMFA: %v", err)
	}
	if resp.Secret == "" {
		t.Error("expected non-empty TOTP secret")
	}
	if !strings.HasPrefix(resp.OTPAuthURL, "otpauth://totp/") {
		t.Errorf("expected otpauth:// URL, got %q", resp.OTPAuthURL)
	}
	if len(resp.BackupCodes) != 8 {
		t.Errorf("expected 8 backup codes, got %d", len(resp.BackupCodes))
	}
	// All 8 backup codes must be unique and 8 chars each
	seen := make(map[string]bool)
	for i, code := range resp.BackupCodes {
		if len(code) != 8 {
			t.Errorf("backup code %d: length %d, want 8", i, len(code))
		}
		if seen[code] {
			t.Errorf("duplicate backup code at index %d: %q", i, code)
		}
		seen[code] = true
	}
}

func TestServiceMFAEnroll_AlreadyActive(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "mfaactive@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "mfaactive@mfa.com", "TestP@ss1")

	resp, err := svc.EnrollMFA(context.Background(), pair.User.ID, "mfaactive@mfa.com")
	if err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	code, _ := totp.GenerateCode(resp.Secret, time.Now())
	if err := svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code); err != nil {
		t.Fatalf("verify: %v", err)
	}

	_, err = svc.EnrollMFA(context.Background(), pair.User.ID, "mfaactive@mfa.com")
	if err == nil {
		t.Fatal("expected ErrMFAAlreadyActive on second enroll")
	}
}

func TestServiceMFAEnroll_ClearsOldPending(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "repend@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "repend@mfa.com", "TestP@ss1")

	// Start enrollment but do not verify
	if _, err := svc.EnrollMFA(context.Background(), pair.User.ID, "repend@mfa.com"); err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	// Re-enroll before verifying — must succeed by clearing the previous pending secret
	resp, err := svc.EnrollMFA(context.Background(), pair.User.ID, "repend@mfa.com")
	if err != nil {
		t.Fatalf("re-enroll after abandoned pending: %v", err)
	}
	if resp.Secret == "" {
		t.Error("expected fresh secret from re-enrollment")
	}
}

func TestServiceMFAVerifyEnrollment_ValidCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "verifyen@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "verifyen@mfa.com", "TestP@ss1")

	resp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "verifyen@mfa.com")
	code, err := totp.GenerateCode(resp.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if err := svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code); err != nil {
		t.Fatalf("VerifyMFAEnrollment: %v", err)
	}
}

func TestServiceMFAVerifyEnrollment_InvalidCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "badcode@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "badcode@mfa.com", "TestP@ss1")

	_, _ = svc.EnrollMFA(context.Background(), pair.User.ID, "badcode@mfa.com")

	if err := svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, "000000"); err == nil {
		t.Fatal("expected error for wrong TOTP code")
	}
}

func TestServiceMFAVerifyEnrollment_NoPendingSecret(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "nopend@svc.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "nopend@svc.com", "TestP@ss1")

	if err := svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, "123456"); err == nil {
		t.Fatal("expected error when no pending enrollment")
	}
}

func TestServiceMFAChallenge_ValidTOTP(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "challenge@mfa.com", "TestP@ss1", true)

	pair := mustLogin(t, svc, "challenge@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "challenge@mfa.com")
	activateCode, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, activateCode)

	loginResult, err := svc.Login(context.Background(), "challenge@mfa.com", "TestP@ss1", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !loginResult.MFARequired {
		t.Fatal("expected MFA required after enrollment")
	}
	if loginResult.MFAToken == "" {
		t.Fatal("expected mfa_token")
	}
	if loginResult.Pair != nil {
		t.Fatal("token pair must be nil until MFA challenge is passed")
	}

	challengeCode, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	result, err := svc.VerifyMFAChallenge(context.Background(), loginResult.MFAToken, challengeCode, "127.0.0.1")
	if err != nil {
		t.Fatalf("VerifyMFAChallenge: %v", err)
	}
	if result.Pair == nil || result.Pair.AccessToken == "" {
		t.Fatal("expected full token pair after valid TOTP challenge")
	}
}

func TestServiceMFAChallenge_InvalidCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "badc@mfa.com", "TestP@ss1", true)

	pair := mustLogin(t, svc, "badc@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "badc@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	loginResult, _ := svc.Login(context.Background(), "badc@mfa.com", "TestP@ss1", "127.0.0.1", "test")

	_, err := svc.VerifyMFAChallenge(context.Background(), loginResult.MFAToken, "000000", "127.0.0.1")
	if err == nil {
		t.Fatal("expected error for wrong code")
	}
}

func TestServiceMFAChallenge_ValidBackupCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "bkup@mfa.com", "TestP@ss1", true)

	pair := mustLogin(t, svc, "bkup@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "bkup@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	loginResult, _ := svc.Login(context.Background(), "bkup@mfa.com", "TestP@ss1", "127.0.0.1", "test")
	result, err := svc.VerifyMFAChallenge(context.Background(), loginResult.MFAToken, enrollResp.BackupCodes[0], "127.0.0.1")
	if err != nil {
		t.Fatalf("backup code challenge: %v", err)
	}
	if result.Pair == nil || result.Pair.AccessToken == "" {
		t.Fatal("expected full token pair after backup code")
	}
}

func TestServiceMFAChallenge_BackupCode_UsedTwice(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "bkup2@mfa.com", "TestP@ss1", true)

	pair := mustLogin(t, svc, "bkup2@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "bkup2@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	// First use of backup code
	lr1, _ := svc.Login(context.Background(), "bkup2@mfa.com", "TestP@ss1", "127.0.0.1", "test")
	_, err := svc.VerifyMFAChallenge(context.Background(), lr1.MFAToken, enrollResp.BackupCodes[1], "127.0.0.1")
	if err != nil {
		t.Fatalf("first backup use: %v", err)
	}

	// Second use of same backup code must fail
	lr2, _ := svc.Login(context.Background(), "bkup2@mfa.com", "TestP@ss1", "127.0.0.1", "test")
	_, err = svc.VerifyMFAChallenge(context.Background(), lr2.MFAToken, enrollResp.BackupCodes[1], "127.0.0.1")
	if err == nil {
		t.Fatal("expected error: used backup code must be rejected on second use")
	}
}

func TestServiceMFAChallenge_NonMFAJWT(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "nonmfajwt@mfa.com", "TestP@ss1", true)

	// A normal access token (MFAPending=false) must be rejected by VerifyMFAChallenge
	pair := mustLogin(t, svc, "nonmfajwt@mfa.com", "TestP@ss1")
	_, err := svc.VerifyMFAChallenge(context.Background(), pair.AccessToken, "123456", "127.0.0.1")
	if err == nil {
		t.Fatal("expected error: regular access token is not a valid MFA token")
	}
}

func TestServiceDisableMFA_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "disman@mfa.com", "TestP@ss1", true)

	pair := mustLogin(t, svc, "disman@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "disman@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	if err := svc.DisableMFA(context.Background(), pair.User.ID, "TestP@ss1", "127.0.0.1"); err != nil {
		t.Fatalf("DisableMFA: %v", err)
	}

	// Login after disable must return a full token pair without MFA gate
	result, err := svc.Login(context.Background(), "disman@mfa.com", "TestP@ss1", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("login after disable: %v", err)
	}
	if result.MFARequired {
		t.Error("MFA must not be required after disabling")
	}
	if result.Pair == nil {
		t.Error("expected full token pair after MFA is disabled")
	}
}

func TestServiceDisableMFA_WrongPassword(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	seedUser(t, db, "diswrong@mfa.com", "TestP@ss1", true)

	pair := mustLogin(t, svc, "diswrong@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "diswrong@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	if err := svc.DisableMFA(context.Background(), pair.User.ID, "WrongPassword", "127.0.0.1"); err == nil {
		t.Fatal("expected error for wrong password when disabling MFA")
	}
}

// ── HTTP handler MFA tests ────────────────────────────────────────────────────

func TestHandlerMFAStatus_NoMFA(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "statusno@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "statusno@mfa.com", "TestP@ss1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/status", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct{ Active bool `json:"active"` } `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Active {
		t.Error("expected active=false for user without MFA")
	}
}

func TestHandlerMFAStatus_WithActiveMFA(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "statusyes@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "statusyes@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "statusyes@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/status", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var env struct {
		Data struct{ Active bool `json:"active"` } `json:"data"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&env)
	if !env.Data.Active {
		t.Error("expected active=true after MFA enrollment")
	}
}

func TestHandlerMFAStatus_Unauthenticated(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	router, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerMFAEnroll_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "enrhttp@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "enrhttp@mfa.com", "TestP@ss1")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/enroll", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Secret      string   `json:"secret"`
			OTPAuthURL  string   `json:"otpauth_url"`
			BackupCodes []string `json:"backup_codes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Secret == "" {
		t.Error("expected non-empty secret")
	}
	if !strings.HasPrefix(env.Data.OTPAuthURL, "otpauth://") {
		t.Errorf("expected otpauth:// URL, got %q", env.Data.OTPAuthURL)
	}
	if len(env.Data.BackupCodes) != 8 {
		t.Errorf("expected 8 backup codes, got %d", len(env.Data.BackupCodes))
	}
}

func TestHandlerMFAEnroll_AlreadyActive(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "double@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "double@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "double@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/enroll", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestHandlerMFAEnroll_Unauthenticated(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	router, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/enroll", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerMFAVerifyEnroll_ValidCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "verifhttp@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "verifhttp@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "verifhttp@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())

	body := fmt.Sprintf(`{"code":%q}`, code)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify-enroll", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerMFAVerifyEnroll_InvalidCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "invcode@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "invcode@mfa.com", "TestP@ss1")
	_, _ = svc.EnrollMFA(context.Background(), pair.User.ID, "invcode@mfa.com")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify-enroll",
		strings.NewReader(`{"code":"000000"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for wrong code, got %d", rec.Code)
	}
}

func TestHandlerMFAVerifyEnroll_NoPendingEnrollment(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "nopen@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "nopen@mfa.com", "TestP@ss1")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify-enroll",
		strings.NewReader(`{"code":"123456"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409 for no pending enrollment, got %d", rec.Code)
	}
}

func TestHandlerMFAVerifyEnroll_MissingCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "misscd@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "misscd@mfa.com", "TestP@ss1")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify-enroll",
		strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing code, got %d", rec.Code)
	}
}

func TestHandlerMFAVerifyEnroll_Unauthenticated(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	router, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify-enroll",
		strings.NewReader(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerMFAChallenge_FullFlow_TOTP(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "fullflow@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "fullflow@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "fullflow@mfa.com")
	activCode, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, activCode)

	// Step 1: Login returns mfa_required + mfa_token
	loginBody := strings.NewReader(`{"email":"fullflow@mfa.com","password":"TestP@ss1"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	var loginEnv struct {
		Data struct {
			MFARequired bool   `json:"mfa_required"`
			MFAToken    string `json:"mfa_token"`
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginEnv); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if !loginEnv.Data.MFARequired {
		t.Fatal("expected mfa_required=true")
	}
	if loginEnv.Data.MFAToken == "" {
		t.Fatal("expected mfa_token in login response")
	}
	if loginEnv.Data.AccessToken != "" {
		t.Error("access_token must be absent before MFA challenge is completed")
	}
	// refresh_token cookie must not be set yet
	for _, c := range loginRec.Result().Cookies() {
		if c.Name == "refresh_token" {
			t.Error("refresh_token cookie must not be set before MFA is verified")
		}
	}

	// Step 2: Challenge with correct TOTP
	challengeCode, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	challengeBody := fmt.Sprintf(`{"mfa_token":%q,"code":%q}`, loginEnv.Data.MFAToken, challengeCode)
	challengeReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/challenge",
		strings.NewReader(challengeBody))
	challengeReq.Header.Set("Content-Type", "application/json")
	challengeRec := httptest.NewRecorder()
	router.ServeHTTP(challengeRec, challengeReq)

	if challengeRec.Code != http.StatusOK {
		t.Fatalf("challenge: expected 200, got %d: %s", challengeRec.Code, challengeRec.Body.String())
	}
	var challengeEnv struct {
		Data struct {
			AccessToken string `json:"access_token"`
			User        struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(challengeRec.Body).Decode(&challengeEnv); err != nil {
		t.Fatalf("decode challenge: %v", err)
	}
	if challengeEnv.Data.AccessToken == "" {
		t.Error("expected access_token after successful TOTP challenge")
	}
	if challengeEnv.Data.User.Email != "fullflow@mfa.com" {
		t.Errorf("expected email fullflow@mfa.com, got %q", challengeEnv.Data.User.Email)
	}
	// Verify refresh_token cookie IS set now
	var gotRefreshCookie bool
	for _, c := range challengeRec.Result().Cookies() {
		if c.Name == "refresh_token" {
			gotRefreshCookie = true
		}
	}
	if !gotRefreshCookie {
		t.Error("expected refresh_token cookie after successful MFA challenge")
	}
}

func TestHandlerMFAChallenge_FullFlow_BackupCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "bkupflow@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "bkupflow@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "bkupflow@mfa.com")
	activCode, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, activCode)

	loginBody := strings.NewReader(`{"email":"bkupflow@mfa.com","password":"TestP@ss1"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	var loginEnv struct {
		Data struct{ MFAToken string `json:"mfa_token"` } `json:"data"`
	}
	_ = json.NewDecoder(loginRec.Body).Decode(&loginEnv)

	// Use backup code index 4 to verify second-use rejection works on different tests
	backupCode := enrollResp.BackupCodes[4]
	challengeBody := fmt.Sprintf(`{"mfa_token":%q,"code":%q}`, loginEnv.Data.MFAToken, backupCode)
	challengeReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/challenge",
		strings.NewReader(challengeBody))
	challengeReq.Header.Set("Content-Type", "application/json")
	challengeRec := httptest.NewRecorder()
	router.ServeHTTP(challengeRec, challengeReq)

	if challengeRec.Code != http.StatusOK {
		t.Fatalf("expected 200 with backup code, got %d: %s", challengeRec.Code, challengeRec.Body.String())
	}
}

func TestHandlerMFAChallenge_InvalidCode(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "invch@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "invch@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "invch@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)
	lr, _ := svc.Login(context.Background(), "invch@mfa.com", "TestP@ss1", "127.0.0.1", "test")

	body := fmt.Sprintf(`{"mfa_token":%q,"code":"000000"}`, lr.MFAToken)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/challenge", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid code, got %d", rec.Code)
	}
}

func TestHandlerMFAChallenge_MissingFields(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	router, _ := setupHandler(t, db)

	// Missing code field
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/challenge",
		strings.NewReader(`{"mfa_token":"sometok"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing code, got %d", rec.Code)
	}
}

func TestHandlerMFADisable_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "dishttp@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "dishttp@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "dishttp@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/disable",
		strings.NewReader(`{"password":"TestP@ss1"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerMFADisable_WrongPassword(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "diswrhttp@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "diswrhttp@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "diswrhttp@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/disable",
		strings.NewReader(`{"password":"WrongPass"}`))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for wrong disable password, got %d", rec.Code)
	}
}

func TestHandlerMFADisable_Unauthenticated(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	router, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/disable",
		strings.NewReader(`{"password":"any"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerLogin_MFARequired(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)
	router, _ := setupHandler(t, db)

	seedUser(t, db, "mfagate@mfa.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "mfagate@mfa.com", "TestP@ss1")
	enrollResp, _ := svc.EnrollMFA(context.Background(), pair.User.ID, "mfagate@mfa.com")
	code, _ := totp.GenerateCode(enrollResp.Secret, time.Now())
	_ = svc.VerifyMFAEnrollment(context.Background(), pair.User.ID, code)

	body := strings.NewReader(`{"email":"mfagate@mfa.com","password":"TestP@ss1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for MFA-gated login, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			MFARequired bool   `json:"mfa_required"`
			MFAToken    string `json:"mfa_token"`
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !env.Data.MFARequired {
		t.Error("expected mfa_required=true")
	}
	if env.Data.MFAToken == "" {
		t.Error("expected mfa_token")
	}
	if env.Data.AccessToken != "" {
		t.Error("access_token must be absent in the MFA gate response")
	}
}
