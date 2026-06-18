package closure_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/reddybashanaguru/opsowl/internal/closure"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

// ── test infrastructure ───────────────────────────────────────────────────────

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
		t.Fatalf("connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.Up(sqlDB, "../../db/migrations/incremental"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return sqlDB, func() { pool.Close(); _ = pgc.Terminate(ctx) }
}

func newJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return mgr
}

func seedUser(t *testing.T, db *sql.DB, email string, role dbgen.UserRole) uuid.UUID {
	t.Helper()
	hash, _ := auth.HashPassword("TestP@ss1")
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email: email, PasswordHash: hash,
		FullName: string(role), Role: role,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

func seedCase(t *testing.T, db *sql.DB, creatorID uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO cases (case_number, title, description, status, priority, created_by, organization_id, tags)
		VALUES ('OWL-TEST-' || nextval('case_number_seq')::TEXT,
		        'Closure test case', '', 'open', 'high', $1,
		        '00000000-0000-0000-0000-000000000001', '{}')
		RETURNING id`, creatorID).Scan(&id)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return id
}

func bearer(t *testing.T, mgr *auth.JWTManager, id uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := mgr.Sign(id, email, role)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return "Bearer " + tok
}

func newRouter(db *sql.DB, jwtMgr *auth.JWTManager) http.Handler {
	r := chi.NewRouter()
	repo := closure.NewRepository(db)
	svc := closure.NewService(repo, timeline.NoopEmitter{})
	h := closure.NewHandler(svc)
	closure.RegisterRoutes(r, h, jwtMgr)
	return r
}

func decodeData(t *testing.T, body []byte, dst any) {
	t.Helper()
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if err := json.Unmarshal(env.Data, dst); err != nil {
		t.Fatalf("decode data: %v", err)
	}
}

// ── REST contract: GET /cases/{id}/closure ────────────────────────────────────

// When a case exists but no closure record has been created yet, the API must
// return 200 (not 404). Returning 404 would tell clients "this resource doesn't
// exist" — but the closure resource conceptually exists for every case; it just
// hasn't been started. 404 should only fire when the parent case itself is missing.
func TestGetClosure_NotStarted_Returns200WithCaseID(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	jwtMgr := newJWT(t)
	analystID := seedUser(t, db, "analyst@closure.test", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)

	req := httptest.NewRequest(http.MethodGet, "/cases/"+caseID.String()+"/closure", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, analystID, "analyst@closure.test", dbgen.UserRoleAnalyst))
	w := httptest.NewRecorder()
	newRouter(db, jwtMgr).ServeHTTP(w, req)

	// Must be 200 — not 404
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp closure.ClosureResponse
	decodeData(t, w.Body.Bytes(), &resp)

	// status must be "not_started"
	if resp.Status != "not_started" {
		t.Errorf("expected status=not_started, got %q", resp.Status)
	}

	// case_id must be populated — not empty string
	if resp.CaseID != caseID.String() {
		t.Errorf("expected case_id=%s, got %q", caseID, resp.CaseID)
	}
}

// When a closure record has been created, GET must return 200 with status "draft"
// and the correct case_id — not an empty struct.
func TestGetClosure_AfterCreate_Returns200WithDraftStatus(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	jwtMgr := newJWT(t)
	analystID := seedUser(t, db, "analyst2@closure.test", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := bearer(t, jwtMgr, analystID, "analyst2@closure.test", dbgen.UserRoleAnalyst)
	router := newRouter(db, jwtMgr)

	// Seed an approved finding so checklist passes
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO findings (case_id, title, finding_type, description, status, created_by, organization_id)
		VALUES ($1, 'Test finding', 'suspicious_activity', 'structuring detected', 'approved', $2,
		        '00000000-0000-0000-0000-000000000001')`, caseID, analystID)
	if err != nil {
		t.Fatalf("seed finding: %v", err)
	}

	// Create closure
	body, _ := json.Marshal(map[string]any{
		"disposition":         "str_filed",
		"summary":             "Confirmed structuring — STR filed with FIU-IND.",
		"reason_code":         "structuring",
		"filed_str":           true,
		"risk_rating_at_close": "high",
		"sar_considered":      true,
	})
	createReq := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/closure",
		bytes.NewReader(body))
	createReq.Header.Set("Authorization", tok)
	createReq.Header.Set("Content-Type", "application/json")
	cw := httptest.NewRecorder()
	router.ServeHTTP(cw, createReq)
	if cw.Code != http.StatusCreated {
		t.Fatalf("create closure: expected 201, got %d: %s", cw.Code, cw.Body.String())
	}

	// GET must now return 200 with status=draft and populated case_id
	getReq := httptest.NewRequest(http.MethodGet, "/cases/"+caseID.String()+"/closure", nil)
	getReq.Header.Set("Authorization", tok)
	gw := httptest.NewRecorder()
	router.ServeHTTP(gw, getReq)

	if gw.Code != http.StatusOK {
		t.Fatalf("get closure: expected 200, got %d: %s", gw.Code, gw.Body.String())
	}

	var resp closure.ClosureResponse
	decodeData(t, gw.Body.Bytes(), &resp)

	if resp.Status != "draft" {
		t.Errorf("expected status=draft, got %q", resp.Status)
	}
	if resp.CaseID != caseID.String() {
		t.Errorf("expected case_id=%s, got %q", caseID, resp.CaseID)
	}
	if resp.Disposition != "str_filed" {
		t.Errorf("expected disposition=str_filed, got %q", resp.Disposition)
	}
}

// GET /cases/{id}/closure/checklist must always return 200 regardless of
// whether any closure record exists — it is a read-only status summary.
func TestGetClosureChecklist_AlwaysReturns200(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	jwtMgr := newJWT(t)
	analystID := seedUser(t, db, "analyst3@closure.test", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)

	req := httptest.NewRequest(http.MethodGet, "/cases/"+caseID.String()+"/closure/checklist", nil)
	req.Header.Set("Authorization", bearer(t, jwtMgr, analystID, "analyst3@closure.test", dbgen.UserRoleAnalyst))
	w := httptest.NewRecorder()
	newRouter(db, jwtMgr).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Submitting a closure must be blocked when the submitter tries to approve
// their own submission (4-eyes principle required by RBI).
func TestSubmitClosure_BlocksSelfApproval(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()

	jwtMgr := newJWT(t)
	analystID := seedUser(t, db, "analyst4@closure.test", dbgen.UserRoleAnalyst)
	managerID := seedUser(t, db, "mgr@closure.test", dbgen.UserRoleManager)
	caseID := seedCase(t, db, analystID)
	mgrTok := bearer(t, jwtMgr, managerID, "mgr@closure.test", dbgen.UserRoleManager)
	router := newRouter(db, jwtMgr)

	// Seed approved finding
	_, _ = db.ExecContext(context.Background(), `
		INSERT INTO findings (case_id, title, finding_type, description, status, created_by, organization_id)
		VALUES ($1, 'f', 'suspicious_activity', 'd', 'approved', $2, '00000000-0000-0000-0000-000000000001')`,
		caseID, managerID)

	// Create closure as manager
	body, _ := json.Marshal(map[string]any{
		"disposition": "str_filed", "summary": "test summary for self approval",
		"reason_code": "structuring", "filed_str": true,
		"risk_rating_at_close": "high", "sar_considered": true,
	})
	cr := httptest.NewRequest(http.MethodPost, "/cases/"+caseID.String()+"/closure", bytes.NewReader(body))
	cr.Header.Set("Authorization", mgrTok)
	cr.Header.Set("Content-Type", "application/json")
	cw := httptest.NewRecorder()
	router.ServeHTTP(cw, cr)
	if cw.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", cw.Code, cw.Body.String())
	}

	var created closure.ClosureResponse
	decodeData(t, cw.Body.Bytes(), &created)

	// Submit as same manager
	sr := httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/closure/"+created.ID+"/submit", nil)
	sr.Header.Set("Authorization", mgrTok)
	sw := httptest.NewRecorder()
	router.ServeHTTP(sw, sr)
	if sw.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", sw.Code, sw.Body.String())
	}

	// Approve as same manager — must be blocked (409 Conflict)
	ar := httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/closure/"+created.ID+"/approve", nil)
	ar.Header.Set("Authorization", mgrTok)
	aw := httptest.NewRecorder()
	router.ServeHTTP(aw, ar)

	if aw.Code != http.StatusConflict {
		t.Errorf("self-approval: expected 409, got %d: %s", aw.Code, aw.Body.String())
	}
}
