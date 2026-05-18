package sla_test

import (
	"bytes"
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

	"github.com/reddybashanaguru/opsowl/internal/auth"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/sla"
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
	if err := goose.Up(sqlDB, "../../migrations"); err != nil {
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
	h := sla.NewHandler(q)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		sla.RegisterRoutes(r, h, jwtMgr)
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

// ── GET /sla/policies ─────────────────────────────────────────────────────────

func TestListSLAPolicies_SeededDefaults(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	viewerID := seedUser(t, db, "viewer@sla.com", dbgen.UserRoleViewer)
	tok := makeToken(t, jwtMgr, viewerID, "viewer@sla.com", dbgen.UserRoleViewer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sla/policies", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body struct {
		Data []struct {
			Priority     string `json:"priority"`
			BreachHours  int    `json:"breach_hours"`
			WarningHours int    `json:"warning_hours"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}

	// Migration seeds 4 default policies
	if len(body.Data) != 4 {
		t.Errorf("expected 4 seeded SLA policies, got %d", len(body.Data))
	}

	// Verify critical policy has the tightest breach hours
	for _, p := range body.Data {
		if p.Priority == "critical" {
			if p.BreachHours != 4 {
				t.Errorf("critical policy breach_hours: expected 4, got %d", p.BreachHours)
			}
			if p.WarningHours != 3 {
				t.Errorf("critical policy warning_hours: expected 3, got %d", p.WarningHours)
			}
		}
	}
}

func TestListSLAPolicies_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, _ := setupRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sla/policies", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── PUT /sla/policies ─────────────────────────────────────────────────────────

func TestUpsertSLAPolicy_AdminSuccess(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin@sla.com", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin@sla.com", dbgen.UserRoleAdmin)

	body := map[string]interface{}{
		"priority":      "high",
		"warning_hours": 18,
		"breach_hours":  24,
		"warn_pct":      75,
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla/policies", bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			Priority     string `json:"priority"`
			BreachHours  int    `json:"breach_hours"`
			WarningHours int    `json:"warning_hours"`
			WarnPct      int    `json:"warn_pct"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.Data.Priority != "high" {
		t.Errorf("expected priority=high, got %s", resp.Data.Priority)
	}
	if resp.Data.BreachHours != 24 {
		t.Errorf("expected breach_hours=24, got %d", resp.Data.BreachHours)
	}
	if resp.Data.WarnPct != 75 {
		t.Errorf("expected warn_pct=75, got %d", resp.Data.WarnPct)
	}
}

func TestUpsertSLAPolicy_403_AnalystForbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	analystID := seedUser(t, db, "analyst@sla.com", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, analystID, "analyst@sla.com", dbgen.UserRoleAnalyst)

	body := map[string]interface{}{
		"priority":     "medium",
		"breach_hours": 48,
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla/policies", bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestUpsertSLAPolicy_400_MissingFields(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin2@sla.com", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin2@sla.com", dbgen.UserRoleAdmin)

	// Missing priority
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla/policies",
		bytes.NewBufferString(`{"breach_hours": 24}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
