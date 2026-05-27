package analytics_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/reddybashanaguru/opsowl/internal/analytics"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

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

	return sqlDB, func() {
		pool.Close()
		_ = pgc.Terminate(ctx)
	}
}

func seedUser(t *testing.T, db *sql.DB, email string, role dbgen.UserRole) uuid.UUID {
	t.Helper()
	hash, err := auth.HashPassword("TestP@ss1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email:        email,
		PasswordHash: hash,
		FullName:     fmt.Sprintf("%s user", role),
		Role:         role,
	})
	if err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}
	return u.ID
}

func setupRouter(t *testing.T, db *sql.DB) (*chi.Mux, *auth.JWTManager) {
	t.Helper()

	jwtMgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}

	q := dbgen.New(db)
	h := analytics.NewHandler(q)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		analytics.RegisterRoutes(r, h, jwtMgr)
	})
	return r, jwtMgr
}

func makeToken(t *testing.T, jwtMgr *auth.JWTManager, userID uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := jwtMgr.Sign(userID, email, role)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

// ── GET /analytics/workload ───────────────────────────────────────────────────

func TestWorkload_EmptyDatabase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	managerID := seedUser(t, db, "manager@analytics.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, managerID, "manager@analytics.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Should return valid JSON array (possibly empty)
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
}

func TestWorkload_403_AnalystForbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	analystID := seedUser(t, db, "analyst@analytics.com", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, analystID, "analyst@analytics.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for analyst role, got %d", rr.Code)
	}
}

func TestWorkload_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, _ := setupRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── GET /analytics/risk ───────────────────────────────────────────────────────

func TestRiskLeaderboard_Empty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	analystID := seedUser(t, db, "analyst2@analytics.com", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, analystID, "analyst2@analytics.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Should return a valid JSON envelope
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
}

func TestRiskLeaderboard_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, _ := setupRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestRiskLeaderboard_403_ViewerForbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	viewerID := seedUser(t, db, "viewer@analytics.com", dbgen.UserRoleViewer)
	tok := makeToken(t, jwtMgr, viewerID, "viewer@analytics.com", dbgen.UserRoleViewer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer, got %d", rr.Code)
	}
}

func TestRiskLeaderboard_WithLimitParam(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	mgr := seedUser(t, db, "mgr-risk@analytics.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgr, "mgr-risk@analytics.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWorkload_200_Manager(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	mgr := seedUser(t, db, "mgr-wl@analytics.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgr, "mgr-wl@analytics.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ── GET /analytics/workload/export ────────────────────────────────────────────

func TestExportWorkload_200_CSV(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	mgr := seedUser(t, db, "mgr-exp@analytics.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgr, "mgr-exp@analytics.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload/export", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if ct == "" || ct[:8] != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %q", ct)
	}
	cd := rr.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header to be set")
	}
	body := rr.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty CSV body")
	}
	// First line must be the CSV header row
	firstLine := body[:len("full_name,email,role,active_cases,open_tasks,overdue_tasks,avg_case_age_hours")]
	expected := "full_name,email,role,active_cases,open_tasks,overdue_tasks,avg_case_age_hours"
	if firstLine != expected {
		t.Errorf("unexpected CSV header: %q", firstLine)
	}
}

func TestExportWorkload_403_Analyst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst-exp@analytics.com", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst-exp@analytics.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload/export", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestExportWorkload_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, _ := setupRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/workload/export", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── GET /analytics/risk/export ────────────────────────────────────────────────

func TestExportRisk_200_CSV(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	mgr := seedUser(t, db, "mgr-risk-exp@analytics.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgr, "mgr-risk-exp@analytics.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk/export", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if ct == "" || ct[:8] != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %q", ct)
	}
	cd := rr.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header to be set")
	}
	body := rr.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty CSV body")
	}
	expectedHeader := "case_number,title,status,priority,sla_breached,risk_score,risk_level,analysed_at"
	firstLine := body[:len(expectedHeader)]
	if firstLine != expectedHeader {
		t.Errorf("unexpected CSV header: %q", firstLine)
	}
}

func TestExportRisk_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, _ := setupRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk/export", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestExportRisk_WithLimitParam(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	mgr := seedUser(t, db, "mgr-risk-lim@analytics.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgr, "mgr-risk-lim@analytics.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk/export?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExportRisk_403_Viewer(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	vID := seedUser(t, db, "viewer-exp@analytics.com", dbgen.UserRoleViewer)
	tok := makeToken(t, jwtMgr, vID, "viewer-exp@analytics.com", dbgen.UserRoleViewer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/risk/export", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
