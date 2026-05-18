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

	"go.uber.org/zap"

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

// ── Scheduler / MarkBreachedCases ─────────────────────────────────────────────

// seedCase inserts a raw case with a controlled created_at so tests can simulate
// cases that are older than the TAT breach threshold.
func seedCase(t *testing.T, sqlDB *sql.DB, userID uuid.UUID, priority, status string, createdAt time.Time) uuid.UUID {
	t.Helper()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	id := uuid.New()
	// Build case_number in Go to avoid pgx type-deduction conflict when $1
	// (uuid) appears in both the id column and a string expression.
	caseNum := "TST-" + id.String()[:8]
	_, err := sqlDB.ExecContext(context.Background(), `
		INSERT INTO cases
		  (id, case_number, title, description, status, priority,
		   created_by, organization_id, sla_breached, created_at, updated_at, tags)
		VALUES
		  ($1, $2, 'Test Case', '', $3::case_status, $4::case_priority,
		   $5, $6, false, $7, $7, '{}')`,
		id, caseNum, status, priority, userID, orgID, createdAt,
	)
	if err != nil {
		t.Fatalf("seedCase: %v", err)
	}
	return id
}

// TestMarkBreachedCases_FlagsOverdueCase verifies that a critical case older
// than the 4-hour breach threshold is correctly marked sla_breached=true.
func TestMarkBreachedCases_FlagsOverdueCase(t *testing.T) {
	sqlDB, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, sqlDB, "analyst@sweep.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, sqlDB, userID, "critical", "open", time.Now().Add(-5*time.Hour))

	q := dbgen.New(sqlDB)
	n, err := q.MarkBreachedCases(context.Background())
	if err != nil {
		t.Fatalf("MarkBreachedCases: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 newly breached, got %d", n)
	}

	var breached bool
	if err := sqlDB.QueryRowContext(context.Background(),
		"SELECT sla_breached FROM cases WHERE id = $1", caseID,
	).Scan(&breached); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !breached {
		t.Error("expected sla_breached=true, got false")
	}
}

// TestMarkBreachedCases_SkipsRecentCase ensures a case created just now
// (well within the 4h TAT) is NOT flagged.
func TestMarkBreachedCases_SkipsRecentCase(t *testing.T) {
	sqlDB, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, sqlDB, "analyst@recent.com", dbgen.UserRoleAnalyst)
	seedCase(t, sqlDB, userID, "critical", "open", time.Now().Add(-1*time.Hour))

	q := dbgen.New(sqlDB)
	n, err := q.MarkBreachedCases(context.Background())
	if err != nil {
		t.Fatalf("MarkBreachedCases: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 breaches for a recent case, got %d", n)
	}
}

// TestMarkBreachedCases_SkipsClosedCase verifies that closed cases are never
// flagged even if they are very old.
func TestMarkBreachedCases_SkipsClosedCase(t *testing.T) {
	sqlDB, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, sqlDB, "analyst@closed.com", dbgen.UserRoleAnalyst)
	seedCase(t, sqlDB, userID, "critical", "closed", time.Now().Add(-48*time.Hour))

	q := dbgen.New(sqlDB)
	n, err := q.MarkBreachedCases(context.Background())
	if err != nil {
		t.Fatalf("MarkBreachedCases: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 breaches for a closed case, got %d", n)
	}
}

// TestMarkBreachedCases_Idempotent verifies that running the sweep twice on
// the same already-breached case returns 0 on the second run.
func TestMarkBreachedCases_Idempotent(t *testing.T) {
	sqlDB, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, sqlDB, "analyst@idem.com", dbgen.UserRoleAnalyst)
	seedCase(t, sqlDB, userID, "critical", "open", time.Now().Add(-10*time.Hour))

	q := dbgen.New(sqlDB)

	n1, err := q.MarkBreachedCases(context.Background())
	if err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	if n1 != 1 {
		t.Errorf("first sweep: expected 1, got %d", n1)
	}

	n2, err := q.MarkBreachedCases(context.Background())
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if n2 != 0 {
		t.Errorf("second sweep (idempotent): expected 0, got %d", n2)
	}
}

// TestMarkBreachedCases_MultiPriority checks all four priority tiers against
// their distinct breach thresholds from the seeded sla_policies.
//
//	critical = 4h, high = 24h, medium = 72h, low = 168h
func TestMarkBreachedCases_MultiPriority(t *testing.T) {
	sqlDB, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, sqlDB, "analyst@multi.com", dbgen.UserRoleAnalyst)

	tests := []struct {
		priority     string
		age          time.Duration
		expectBreach bool
	}{
		// just over threshold
		{"critical", 5 * time.Hour, true},
		{"high", 25 * time.Hour, true},
		{"medium", 73 * time.Hour, true},
		{"low", 169 * time.Hour, true},
		// just under threshold — should not breach
		{"critical", 3 * time.Hour, false},
		{"high", 20 * time.Hour, false},
	}

	ids := make([]uuid.UUID, len(tests))
	for i, tc := range tests {
		ids[i] = seedCase(t, sqlDB, userID, tc.priority, "open", time.Now().Add(-tc.age))
	}

	q := dbgen.New(sqlDB)
	_, err := q.MarkBreachedCases(context.Background())
	if err != nil {
		t.Fatalf("MarkBreachedCases: %v", err)
	}

	for i, tc := range tests {
		var got bool
		if err := sqlDB.QueryRowContext(context.Background(),
			"SELECT sla_breached FROM cases WHERE id = $1", ids[i],
		).Scan(&got); err != nil {
			t.Fatalf("scan case %s: %v", tc.priority, err)
		}
		if got != tc.expectBreach {
			t.Errorf("priority=%s age=%v: expected sla_breached=%v, got %v",
				tc.priority, tc.age, tc.expectBreach, got)
		}
	}
}

// TestScheduler_StartExitsOnContextCancel starts the scheduler, cancels the
// context, and confirms the goroutine exits cleanly without hanging.
func TestScheduler_StartExitsOnContextCancel(t *testing.T) {
	sqlDB, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, sqlDB, "analyst@exit.com", dbgen.UserRoleAnalyst)
	// Seed an overdue case so the initial sweep has work to do.
	seedCase(t, sqlDB, userID, "critical", "open", time.Now().Add(-6*time.Hour))

	q := dbgen.New(sqlDB)
	sched := sla.NewScheduler(q, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		sched.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// good
	case <-time.After(3 * time.Second):
		t.Error("Scheduler.Start did not exit after context cancellation")
	}

	// Verify the initial sweep ran before the cancel was processed.
	var breached bool
	if err := sqlDB.QueryRowContext(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM cases WHERE sla_breached = true)",
	).Scan(&breached); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !breached {
		t.Error("expected initial sweep to have marked at least one case")
	}
}

func TestUpsertSLAPolicy_400_InvalidJSON(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin3@sla.com", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin3@sla.com", dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla/policies",
		bytes.NewBufferString(`not-json`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpsertSLAPolicy_400_ZeroBreachHours(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin4@sla.com", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin4@sla.com", dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla/policies",
		bytes.NewBufferString(`{"priority":"low","breach_hours":0}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for zero breach_hours, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestListSLAPolicies_200_Manager(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr@sla.com", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr@sla.com", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sla/policies", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}
