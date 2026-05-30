package playbooks_test

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
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/playbooks"
)

// ── test infra ────────────────────────────────────────────────────────────────

// decodeData unwraps the standard {"data": ...} API envelope.
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
	return sqlDB, func() {
		pool.Close()
		_ = pgc.Terminate(ctx)
	}
}

func newJWT(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt: %v", err)
	}
	return mgr
}

func seedAdmin(t *testing.T, db *sql.DB) (uuid.UUID, string) {
	t.Helper()
	hash, _ := auth.HashPassword("TestP@ss1")
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email:        "admin@test.com",
		PasswordHash: hash,
		FullName:     "Admin User",
		Role:         dbgen.UserRoleAdmin,
	})
	if err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	return u.ID, u.Email
}

func seedAnalyst(t *testing.T, db *sql.DB) (uuid.UUID, string) {
	t.Helper()
	hash, _ := auth.HashPassword("TestP@ss1")
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email:        "analyst@test.com",
		PasswordHash: hash,
		FullName:     "Analyst User",
		Role:         dbgen.UserRoleAnalyst,
	})
	if err != nil {
		t.Fatalf("seed analyst: %v", err)
	}
	return u.ID, u.Email
}

func bearer(t *testing.T, mgr *auth.JWTManager, id uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := mgr.Sign(id, email, role)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return "Bearer " + tok
}

func newRouter(db *sql.DB, jwtMgr *auth.JWTManager) http.Handler {
	r := chi.NewRouter()
	repo := playbooks.NewRepository(db)
	svc := playbooks.NewService(repo)
	h := playbooks.NewHandler(svc)
	playbooks.RegisterRoutes(r, h, jwtMgr)
	return r
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestListPlaybooks_Empty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	adminID, adminEmail := seedAdmin(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, adminID, adminEmail, dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/playbooks", nil)
	req.Header.Set("Authorization", tok)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var list []playbooks.PlaybookResponse
	decodeData(t, w.Body.Bytes(), &list)
	if list == nil {
		t.Error("data should not be nil")
	}
}

func TestCreatePlaybook_Admin(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	adminID, adminEmail := seedAdmin(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, adminID, adminEmail, dbgen.UserRoleAdmin)

	body, _ := json.Marshal(map[string]any{
		"name":        "AML STR Playbook",
		"description": "Standard playbook for STR investigations",
		"case_type":   "aml_str",
		"steps": []map[string]any{
			{"title": "Freeze accounts", "priority": "high", "step_order": 1, "due_hours": 4},
			{"title": "Notify compliance", "priority": "medium", "step_order": 2, "due_hours": 24, "depends_on_step": 1},
			{"title": "File STR with FIU", "priority": "high", "step_order": 3, "due_hours": 72, "depends_on_step": 2},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/playbooks", bytes.NewReader(body))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var pb playbooks.PlaybookResponse
	decodeData(t, w.Body.Bytes(), &pb)
	if pb.Name != "AML STR Playbook" {
		t.Errorf("name mismatch: %q", pb.Name)
	}
	if pb.CaseType != "aml_str" {
		t.Errorf("case_type mismatch: %q", pb.CaseType)
	}
	if len(pb.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(pb.Steps))
	}
	if len(pb.Steps) > 1 && pb.Steps[1].DependsOnStep != 1 {
		t.Errorf("step 2 depends_on_step: expected 1, got %d", pb.Steps[1].DependsOnStep)
	}
}

func TestCreatePlaybook_Analyst_Forbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	analystID, analystEmail := seedAnalyst(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, analystID, analystEmail, dbgen.UserRoleAnalyst)

	body, _ := json.Marshal(map[string]any{
		"name":      "Should Fail",
		"case_type": "fraud",
		"steps":     []any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/playbooks", bytes.NewReader(body))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePlaybook_MissingName(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	adminID, adminEmail := seedAdmin(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, adminID, adminEmail, dbgen.UserRoleAdmin)

	body, _ := json.Marshal(map[string]any{
		"case_type": "fraud",
		"steps":     []any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/playbooks", bytes.NewReader(body))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPlaybook_NotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	adminID, adminEmail := seedAdmin(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, adminID, adminEmail, dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/playbooks/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", tok)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdatePlaybook_Deactivate(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	adminID, adminEmail := seedAdmin(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, adminID, adminEmail, dbgen.UserRoleAdmin)

	// Create
	createBody, _ := json.Marshal(map[string]any{
		"name": "Fraud Playbook", "case_type": "fraud", "steps": []any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/playbooks", bytes.NewReader(createBody))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d", w.Code)
	}
	var created playbooks.PlaybookResponse
	decodeData(t, w.Body.Bytes(), &created)

	// Deactivate
	patchBody, _ := json.Marshal(map[string]any{"is_active": false})
	req2 := httptest.NewRequest(http.MethodPatch, "/playbooks/"+created.ID.String(), bytes.NewReader(patchBody))
	req2.Header.Set("Authorization", tok)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("patch failed: %d: %s", w2.Code, w2.Body.String())
	}
	var updated playbooks.PlaybookResponse
	decodeData(t, w2.Body.Bytes(), &updated)
	if updated.IsActive {
		t.Error("expected is_active=false after deactivate")
	}
}

func TestDeletePlaybook(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	jwtMgr := newJWT(t)
	adminID, adminEmail := seedAdmin(t, db)
	router := newRouter(db, jwtMgr)
	tok := bearer(t, jwtMgr, adminID, adminEmail, dbgen.UserRoleAdmin)

	// Create
	createBody, _ := json.Marshal(map[string]any{
		"name": "KYC Playbook", "case_type": "kyc", "steps": []any{},
	})
	req := httptest.NewRequest(http.MethodPost, "/playbooks", bytes.NewReader(createBody))
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var created playbooks.PlaybookResponse
	decodeData(t, w.Body.Bytes(), &created)

	// Delete
	req2 := httptest.NewRequest(http.MethodDelete, "/playbooks/"+created.ID.String(), nil)
	req2.Header.Set("Authorization", tok)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w2.Code, w2.Body.String())
	}
}
