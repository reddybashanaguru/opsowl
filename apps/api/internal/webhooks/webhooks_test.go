package webhooks_test

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
	"github.com/reddybashanaguru/opsowl/internal/webhooks"
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

func setupRouter(t *testing.T, db *sql.DB) (*chi.Mux, *auth.JWTManager) {
	t.Helper()
	jwtMgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}
	q := dbgen.New(db)
	logger := zap.NewNop()
	svc := webhooks.NewService(q, logger)
	h := webhooks.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		webhooks.RegisterRoutes(r, h, jwtMgr)
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
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return bytes.NewBuffer(b)
}

// ── GET /webhooks/event-types ──────────────────────────────────────────────────

func TestListEventTypes_200(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin@wh.test", dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/event-types", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
}

func TestListEventTypes_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/event-types", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rr.Code)
	}
}

func TestListEventTypes_403_Analyst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst@wh.test", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst@wh.test", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/event-types", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 got %d", rr.Code)
	}
}

// ── POST /webhooks (Create) ────────────────────────────────────────────────────

func TestCreate_201(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin2@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin2@wh.test", dbgen.UserRoleAdmin)

	body := jsonBody(t, map[string]any{
		"name":       "Risk Engine",
		"url":        "https://risk.example.com/hook",
		"event_types": []string{"case.created", "case.closed"},
		"is_active":  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d: %s", rr.Code, rr.Body)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestCreate_400_NonHTTPS(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin3@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin3@wh.test", dbgen.UserRoleAdmin)

	body := jsonBody(t, map[string]any{
		"name": "Bad Hook", "url": "http://insecure.example.com/hook",
		"event_types": []string{"case.created"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-https URL, got %d", rr.Code)
	}
}

func TestCreate_400_ShortSecret(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin4@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin4@wh.test", dbgen.UserRoleAdmin)

	body := jsonBody(t, map[string]any{
		"name": "Short Secret", "url": "https://ok.example.com/hook",
		"signing_secret": "tooshort",
		"event_types":    []string{"case.created"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short secret, got %d", rr.Code)
	}
}

func TestCreate_403_Manager(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr@wh.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr@wh.test", dbgen.UserRoleManager)

	body := jsonBody(t, map[string]any{
		"name": "Mgr Hook", "url": "https://ok.example.com/hook",
		"event_types": []string{"case.created"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for manager, got %d", rr.Code)
	}
}

// ── GET /webhooks ──────────────────────────────────────────────────────────────

func TestList_200(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin5@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin5@wh.test", dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
}

// ── GET + PUT + DELETE lifecycle ───────────────────────────────────────────────

func createWebhook(t *testing.T, r http.Handler, tok string) string {
	t.Helper()
	body := jsonBody(t, map[string]any{
		"name": "Lifecycle Hook", "url": "https://lifecycle.example.com/hook",
		"event_types": []string{"case.created"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("createWebhook: expected 201 got %d: %s", rr.Code, rr.Body)
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
	adminID := seedUser(t, db, "admin6@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin6@wh.test", dbgen.UserRoleAdmin)

	id := createWebhook(t, r, tok)

	// GET by ID
	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET: expected 200 got %d", rr.Code)
	}

	// PUT update — must include a valid signing_secret (≥16 chars)
	active := false
	upd := jsonBody(t, map[string]any{
		"name": "Updated Hook", "url": "https://updated.example.com/hook",
		"signing_secret": "updated-secret-key-32-bytes-long!",
		"event_types":    []string{"case.closed"}, "is_active": active,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/webhooks/"+id, upd)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200 got %d: %s", rr.Code, rr.Body)
	}

	// DELETE
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/webhooks/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204 got %d", rr.Code)
	}

	// GET after delete → 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr.Code)
	}
}

func TestGet_404_UnknownID(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin7@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin7@wh.test", dbgen.UserRoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 got %d", rr.Code)
	}
}

// ── GET /webhooks/{id}/deliveries ──────────────────────────────────────────────

func TestListDeliveries_200(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	adminID := seedUser(t, db, "admin8@wh.test", dbgen.UserRoleAdmin)
	tok := makeToken(t, jwtMgr, adminID, "admin8@wh.test", dbgen.UserRoleAdmin)

	id := createWebhook(t, r, tok)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/"+id+"/deliveries", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
}

// ── GenerateSecret ────────────────────────────────────────────────────────────

func TestGenerateSecret_Length(t *testing.T) {
	secret, err := webhooks.GenerateSecret()
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}
	if len(secret) < 32 {
		t.Errorf("expected secret length ≥ 32, got %d", len(secret))
	}
}
