package auth_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if err := goose.Up(sqlDB, "../../migrations"); err != nil {
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
			pair, err := svc.Login(context.Background(), tc.email, tc.pass, "127.0.0.1", "go-test")
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pair.AccessToken == "" {
				t.Fatal("expected access token")
			}
			if !strings.HasPrefix(pair.RefreshToken, "rt_") {
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
	pair, err := svc.Login(context.Background(), "refresh@test.com", "SecureP@ss1", "127.0.0.1", "go-test")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

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
		p2, _ := svc.Login(context.Background(), "refresh@test.com", "SecureP@ss1", "127.0.0.1", "go-test")
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
	pair, _ := svc.Login(context.Background(), "jwt@test.com", "SecureP@ss1", "127.0.0.1", "go-test")

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
	pair, _ := svc.Login(context.Background(), "viewer@test.com", "SecureP@ss1", "127.0.0.1", "go-test")

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
	pair, _ := svc.Login(context.Background(), "pwd@test.com", "SecureP@ss1", "127.0.0.1", "go-test")

	t.Run("wrong current password", func(t *testing.T) {
		err := svc.ChangePassword(context.Background(), pair.User.ID, "wrongpass", "NewSecureP@ss2", "127.0.0.1")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// Compile-time check: db package used
var _ = fmt.Sprintf
