package auth_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// testDB spins up a real Postgres container, runs migrations, returns sqlDB + cleanup.
func testDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.Run(ctx,
		"pgvector/pgvector:pg17",
		postgres.WithDatabase("opsowl_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "../../db/migrations/incremental"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_ = pgc.Terminate(ctx)
	}
	return sqlDB, cleanup
}

func newTestService(t *testing.T, db *sql.DB) (*auth.Service, *auth.JWTManager) {
	t.Helper()
	q := dbgen.New(db)
	repo := auth.NewRepository(q)
	jwtMgr, err := auth.NewJWTManager(
		"../../keys/private.pem",
		"../../keys/public.pem",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("new jwt manager: %v", err)
	}
	svc := auth.NewService(repo, jwtMgr)
	return svc, jwtMgr
}

func seedUser(t *testing.T, db *sql.DB, email, password string, active bool) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	role := dbgen.UserRoleAnalyst
	_, err = dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		FullName:     "Test User",
		Role:         role,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if !active {
		_, err = db.ExecContext(context.Background(),
			"UPDATE users SET is_active = false WHERE email = $1", email)
		if err != nil {
			t.Fatalf("deactivate user: %v", err)
		}
	}
}

// mustLogin is a test helper that unwraps LoginResult.Pair and fails if MFA is required or an error occurs.
func mustLogin(t *testing.T, svc *auth.Service, email, password string) *auth.TokenPair {
	t.Helper()
	result, err := svc.Login(context.Background(), email, password, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if result.Pair == nil {
		t.Fatal("expected token pair, got MFA required")
	}
	return result.Pair
}

func TestLogin(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)

	seedUser(t, db, "analyst@test.com", "SecureP@ss1", true)
	seedUser(t, db, "inactive@test.com", "SecureP@ss1", false)

	tests := []struct {
		name    string
		email   string
		pass    string
		wantErr error
	}{
		{"valid credentials", "analyst@test.com", "SecureP@ss1", nil},
		{"wrong password", "analyst@test.com", "wrongpass", auth.ErrInvalidCredentials},
		{"unknown email", "nobody@test.com", "any", auth.ErrInvalidCredentials},
		{"disabled account", "inactive@test.com", "SecureP@ss1", auth.ErrAccountDisabled},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Login(context.Background(), tc.email, tc.pass, "127.0.0.1", "go-test")
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Pair == nil {
				t.Fatal("expected token pair, got MFA required")
			}
			if result.Pair.AccessToken == "" {
				t.Fatal("expected access token")
			}
			if !strings.HasPrefix(result.Pair.RefreshToken, "rt_") {
				t.Fatal("expected refresh token with rt_ prefix")
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)

	seedUser(t, db, "refresh@test.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "refresh@test.com", "SecureP@ss1")

	t.Run("valid refresh token rotates", func(t *testing.T) {
		newPair, err := svc.Refresh(context.Background(), pair.RefreshToken, "127.0.0.1")
		if err != nil {
			t.Fatalf("refresh: %v", err)
		}
		if newPair.AccessToken == "" || newPair.RefreshToken == "" {
			t.Fatal("expected new token pair")
		}
		if newPair.RefreshToken == pair.RefreshToken {
			t.Fatal("refresh token should rotate")
		}
		// old token must be revoked
		_, err = svc.Refresh(context.Background(), pair.RefreshToken, "127.0.0.1")
		if err == nil {
			t.Fatal("expected error using revoked token")
		}
	})

	t.Run("revoked token triggers family revocation", func(t *testing.T) {
		// Login fresh
		p2 := mustLogin(t, svc, "refresh@test.com", "SecureP@ss1")
		// Use once to rotate
		p3, _ := svc.Refresh(context.Background(), p2.RefreshToken, "127.0.0.1")
		// Present old (revoked) token — should trigger reuse detection
		_, err := svc.Refresh(context.Background(), p2.RefreshToken, "127.0.0.1")
		if err == nil {
			t.Fatal("expected reuse detection error")
		}
		// Family revocation is async (goroutine) — wait for it to complete
		time.Sleep(100 * time.Millisecond)
		// New token should also be revoked now (family revocation)
		_, err = svc.Refresh(context.Background(), p3.RefreshToken, "127.0.0.1")
		if err == nil {
			t.Fatal("expected family-revoked token to fail")
		}
	})
}

func TestJWTValidateMiddleware(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)

	seedUser(t, db, "jwt@test.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "jwt@test.com", "SecureP@ss1")

	mw := auth.JWTValidate(jwtMgr)

	t.Run("valid token passes", func(t *testing.T) {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			claims := auth.ClaimsFromCtx(r.Context())
			if claims == nil {
				t.Error("expected claims in context")
			}
			w.WriteHeader(http.StatusOK)
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		rec := httptest.NewRecorder()
		mw(next).ServeHTTP(rec, req)
		if !called {
			t.Fatal("handler should have been called")
		}
	})

	t.Run("missing header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("tampered token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken+"tampered")
		rec := httptest.NewRecorder()
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})
}

func TestRBACMiddleware(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)

	seedUser(t, db, "viewer@test.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "viewer@test.com", "SecureP@ss1")

	t.Run("viewer denied admin route", func(t *testing.T) {
		jwtMW := auth.JWTValidate(jwtMgr)
		rbacMW := auth.RequireRole(dbgen.UserRoleAdmin, dbgen.UserRoleSuperAdmin)

		handler := jwtMW(rbacMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})
}

func TestChangePassword(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)

	seedUser(t, db, "pwd@test.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "pwd@test.com", "SecureP@ss1")

	t.Run("wrong current password", func(t *testing.T) {
		err := svc.ChangePassword(context.Background(), pair.User.ID, "wrongpass", "NewSecureP@ss2", "127.0.0.1")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// Compile-time check: db package used
var _ = fmt.Sprintf

// ── HTTP Handler tests ────────────────────────────────────────────────────────

func setupHandler(t *testing.T, db *sql.DB) (http.Handler, *auth.JWTManager) {
	t.Helper()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})
	return r, jwtMgr
}

func TestHandlerLogin_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)
	seedUser(t, db, "login@handler.com", "SecureP@ss1", true)

	body := strings.NewReader(`{"email":"login@handler.com","password":"SecureP@ss1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			AccessToken string `json:"access_token"`
			User        struct {
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.AccessToken == "" {
		t.Error("expected access_token in response")
	}
	if env.Data.User.Email != "login@handler.com" {
		t.Errorf("expected email login@handler.com, got %s", env.Data.User.Email)
	}
	// Verify refresh_token cookie was set
	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
		}
	}
	if !found {
		t.Error("expected refresh_token cookie to be set")
	}
}

func TestHandlerLogin_InvalidCredentials(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)
	seedUser(t, db, "creds@handler.com", "SecureP@ss1", true)

	body := strings.NewReader(`{"email":"creds@handler.com","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerLogin_MissingFields(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	body := strings.NewReader(`{"email":"only@handler.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerMe(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "me@handler.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "me@handler.com", "SecureP@ss1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Email != "me@handler.com" {
		t.Errorf("expected me@handler.com, got %s", env.Data.Email)
	}
}

func TestHandlerMe_Unauthorized(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerLogout(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "logout@handler.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "logout@handler.com", "SecureP@ss1")

	body := strings.NewReader(fmt.Sprintf(`{"refresh_token":%q}`, pair.RefreshToken))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", body)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	// Verify cookie is expired
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" && c.MaxAge >= 0 {
			t.Error("expected refresh_token cookie to be expired (MaxAge < 0)")
		}
	}
}

func TestHandlerHandlerRefresh(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "refresh2@handler.com", "SecureP@ss1", true)
	pair := mustLogin(t, svc, "refresh2@handler.com", "SecureP@ss1")

	body := strings.NewReader(fmt.Sprintf(`{"refresh_token":%q}`, pair.RefreshToken))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct{ AccessToken string `json:"access_token"` } `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.AccessToken == "" {
		t.Error("expected new access_token")
	}
}

// ── ChangePassword ────────────────────────────────────────────────────────────

func TestHandlerChangePassword_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "chpwd@handler.com", "OldP@ss12", true)
	pair := mustLogin(t, svc, "chpwd@handler.com", "OldP@ss12")

	body := strings.NewReader(`{"current_password":"OldP@ss12","new_password":"NewP@ss99"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", body)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerChangePassword_WrongCurrentPassword(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "chpwd2@handler.com", "RealP@ss1", true)
	pair := mustLogin(t, svc, "chpwd2@handler.com", "RealP@ss1")

	body := strings.NewReader(`{"current_password":"WrongP@ss","new_password":"NewP@ss99"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", body)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for wrong current password, got %d", rec.Code)
	}
}

func TestHandlerChangePassword_TooShortNewPassword(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "chpwd3@handler.com", "RealP@ss1", true)
	pair := mustLogin(t, svc, "chpwd3@handler.com", "RealP@ss1")

	body := strings.NewReader(`{"current_password":"RealP@ss1","new_password":"short"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", body)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short new password, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerChangePassword_Unauthorized(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	body := strings.NewReader(`{"current_password":"anything","new_password":"NewP@ss99"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ── Refresh error paths ───────────────────────────────────────────────────────

func TestHandlerRefresh_MissingToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRefresh_InvalidToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	body := strings.NewReader(`{"refresh_token":"invalid.garbage.token"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ── VerifyToken ───────────────────────────────────────────────────────────────

func TestServiceVerifyToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)

	seedUser(t, db, "verify@test.com", "TestP@ss1", true)
	// Login to get a valid access token for VerifyToken test
	pair := mustLogin(t, svc, "verify@test.com", "TestP@ss1")
	userID := pair.User.ID

	tok, err := jwtMgr.Sign(userID, "verify@test.com", dbgen.UserRoleAnalyst)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	claims, err := svc.VerifyToken(tok)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, claims.UserID)
	}

	_, err = svc.VerifyToken("invalid.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

// ── Login error paths ─────────────────────────────────────────────────────────

func TestHandlerLogin_InvalidJSON(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerLogin_UserNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	h, _ := setupHandler(t, db)

	body := strings.NewReader(`{"email":"nobody@test.com","password":"TestP@ss1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// ── LDAP unit tests (no real server required) ─────────────────────────────────

func TestLDAPConfig_Enabled(t *testing.T) {
	var nilCfg *auth.LDAPConfig
	if nilCfg.Enabled() {
		t.Error("nil config should not be enabled")
	}

	empty := &auth.LDAPConfig{}
	if empty.Enabled() {
		t.Error("config with empty URL should not be enabled")
	}

	active := &auth.LDAPConfig{URL: "ldap://ldap.example.com:389"}
	if !active.Enabled() {
		t.Error("config with URL should be enabled")
	}
}

func TestNewLDAPAuthenticator(t *testing.T) {
	cfg := &auth.LDAPConfig{
		URL:      "ldap://ldap.example.com:389",
		BaseDN:   "dc=example,dc=com",
		UserAttr: "uid",
	}
	a := auth.NewLDAPAuthenticator(cfg)
	if a == nil {
		t.Error("expected non-nil authenticator")
	}
}

func TestLDAPAuthenticator_EmptyPassword(t *testing.T) {
	cfg := &auth.LDAPConfig{
		URL:      "ldap://localhost:389",
		BaseDN:   "dc=test,dc=com",
		UserAttr: "uid",
	}
	a := auth.NewLDAPAuthenticator(cfg)
	_, err := a.Authenticate("user", "")
	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestWithLDAP_DisabledConfig(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestService(t, db)

	// Disabled config (empty URL) — should be a no-op
	cfg := &auth.LDAPConfig{}
	result := svc.WithLDAP(cfg)
	if result == nil {
		t.Error("WithLDAP should return the service")
	}

	// Login should still use local path
	seedUser(t, db, "ldaptest@test.com", "TestP@ss1", true)
	pair := mustLogin(t, svc, "ldaptest@test.com", "TestP@ss1")
	if pair.AccessToken == "" {
		t.Error("expected access token")
	}
}

// ── Me with last_login set ────────────────────────────────────────────────────

func TestHandlerMe_WithLastLogin(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, jwtMgr := newTestService(t, db)
	h := auth.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, h, jwtMgr)
	})

	seedUser(t, db, "me-ll@handler.com", "TestP@ss1", true)
	// Login first so last_login_at is populated
	pair := mustLogin(t, svc, "me-ll@handler.com", "TestP@ss1")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			LastLoginAt *string `json:"last_login_at"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.LastLoginAt == nil {
		t.Error("expected last_login_at to be populated after login")
	}
}
