package templates_test

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
	"github.com/reddybashanaguru/opsowl/internal/tasks"
	"github.com/reddybashanaguru/opsowl/internal/templates"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

// ── infrastructure ─────────────────────────────────────────────────────────────

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
		t.Fatalf("create pool: %v", err)
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(nil)
	_ = goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, "../../db/migrations/incremental"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return sqlDB, func() {
		pool.Close()
		_ = pgc.Terminate(ctx)
	}
}

type noopEmitter struct{}

func (noopEmitter) Emit(_ context.Context, _ timeline.Event) error { return nil }

func setupRouter(t *testing.T, db *sql.DB) (*chi.Mux, *auth.JWTManager) {
	t.Helper()
	jwtMgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}
	q := dbgen.New(db)
	tasksRepo := tasks.NewRepository(q)
	tasksSvc := tasks.NewService(tasksRepo, noopEmitter{})
	templateSvc := templates.NewService(q, tasksSvc)
	h := templates.NewHandler(templateSvc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		templates.RegisterRoutes(r, h, jwtMgr)
	})
	return r, jwtMgr
}

func seedUser(t *testing.T, db *sql.DB, email string, role dbgen.UserRole) uuid.UUID {
	t.Helper()
	hash, _ := auth.HashPassword("TestP@ss1")
	u, err := dbgen.New(db).CreateUser(context.Background(), dbgen.CreateUserParams{
		Email: email, PasswordHash: hash,
		FullName: fmt.Sprintf("%s user", role), Role: role,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u.ID
}

func seedCase(t *testing.T, db *sql.DB, title string, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	caseNumber := fmt.Sprintf("OWL-TEST-%s", uuid.New().String()[:8])
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO cases (title, case_number, status, priority, organization_id, created_by)
		 VALUES ($1, $2, 'open', 'medium', '00000000-0000-0000-0000-000000000001', $3) RETURNING id`,
		title, caseNumber, createdBy,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return id
}

func makeToken(t *testing.T, jwtMgr *auth.JWTManager, id uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := jwtMgr.Sign(id, email, role)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// ── GET /templates ─────────────────────────────────────────────────────────────

func TestList_200_Authenticated(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst@tmpl.test", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst@tmpl.test", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

func TestList_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── POST /templates (Create) ───────────────────────────────────────────────────

func TestCreate_201_Manager(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr@tmpl.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr@tmpl.test", dbgen.UserRoleManager)

	active := true
	body := jsonBody(t, map[string]any{
		"name":                 "UPI Fraud Template",
		"description_skeleton": "Investigate suspicious UPI transaction.",
		"priority":             "high",
		"tags":                 []string{"upi", "fraud"},
		"is_active":            &active,
		"tasks": []map[string]any{
			{"title": "Verify transaction", "priority": "high", "sort_order": 0},
			{"title": "Contact bank", "priority": "medium", "sort_order": 1},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var env struct {
		Data struct {
			Name  string `json:"name"`
			Tasks []any  `json:"tasks"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Data.Name != "UPI Fraud Template" {
		t.Errorf("unexpected name: %q", env.Data.Name)
	}
	if len(env.Data.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(env.Data.Tasks))
	}
}

func TestCreate_400_MissingName(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr2@tmpl.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr2@tmpl.test", dbgen.UserRoleManager)

	body := jsonBody(t, map[string]any{"name": "", "priority": "medium"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", rr.Code)
	}
}

func TestCreate_403_Analyst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst2@tmpl.test", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst2@tmpl.test", dbgen.UserRoleAnalyst)

	body := jsonBody(t, map[string]any{"name": "Analyst Template"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for analyst, got %d", rr.Code)
	}
}

// ── GET /templates (activeOnly filter) ────────────────────────────────────────

func TestList_ActiveOnly_Filter(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr3@tmpl.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr3@tmpl.test", dbgen.UserRoleManager)

	// Create an inactive template
	inactive := false
	body := jsonBody(t, map[string]any{"name": "Inactive Tmpl", "priority": "low", "is_active": &inactive})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create inactive template: %d", rr.Code)
	}

	// List with active=true — should not include the inactive template
	req = httptest.NewRequest(http.MethodGet, "/api/v1/templates/?active=true", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list active: expected 200, got %d", rr.Code)
	}
	var env struct {
		Data struct {
			Templates []map[string]any `json:"templates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, tmpl := range env.Data.Templates {
		if active, ok := tmpl["is_active"].(bool); ok && !active {
			t.Error("inactive template appeared in active=true list")
		}
	}
}

// ── Full lifecycle: GET / PUT / DELETE ─────────────────────────────────────────

func createTemplate(t *testing.T, r http.Handler, tok string, name string) string {
	t.Helper()
	active := true
	body := jsonBody(t, map[string]any{
		"name": name, "priority": "medium", "is_active": &active,
		"tasks": []map[string]any{{"title": "Step 1", "priority": "medium"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("createTemplate: expected 201, got %d: %s", rr.Code, rr.Body)
	}
	var env struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	return env.Data.ID
}

func TestGetUpdateDelete_Lifecycle(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr4@tmpl.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr4@tmpl.test", dbgen.UserRoleManager)

	id := createTemplate(t, r, tok, "KYC Template")

	// GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", rr.Code)
	}

	// PUT update — replace task stubs
	active := true
	body := jsonBody(t, map[string]any{
		"name": "KYC Template v2", "priority": "high", "is_active": &active,
		"tasks": []map[string]any{
			{"title": "Collect documents", "priority": "high", "sort_order": 0},
			{"title": "Verify address", "priority": "medium", "sort_order": 1},
		},
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/templates/"+id, body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var env struct {
		Data struct {
			Name  string `json:"name"`
			Tasks []any  `json:"tasks"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Data.Name != "KYC Template v2" {
		t.Errorf("expected updated name, got %q", env.Data.Name)
	}
	if len(env.Data.Tasks) != 2 {
		t.Errorf("expected 2 updated tasks, got %d", len(env.Data.Tasks))
	}

	// DELETE
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/templates/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204, got %d", rr.Code)
	}

	// GET after delete → 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestGet_404_Unknown(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst3@tmpl.test", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst3@tmpl.test", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ── POST /templates/{id}/apply ─────────────────────────────────────────────────

func TestApply_200_CreatesTasksOnCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr5@tmpl.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr5@tmpl.test", dbgen.UserRoleManager)

	tmplID := createTemplate(t, r, tok, "Apply Test Template")
	caseID := seedCase(t, db, "Test Case for Apply", mgrID)

	body := jsonBody(t, map[string]any{"case_id": caseID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID+"/apply", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("apply: expected 200, got %d: %s", rr.Code, rr.Body)
	}
	var env struct {
		Data struct{ Status string `json:"status"` } `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Data.Status != "applied" {
		t.Errorf("expected status=applied, got %q", env.Data.Status)
	}
}

func TestApply_400_MissingCaseID(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr6@tmpl.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr6@tmpl.test", dbgen.UserRoleManager)

	tmplID := createTemplate(t, r, tok, "Apply Bad Template")
	body := jsonBody(t, map[string]any{}) // no case_id
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/"+tmplID+"/apply", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing case_id, got %d", rr.Code)
	}
}
