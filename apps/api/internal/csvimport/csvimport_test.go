package csvimport_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"github.com/reddybashanaguru/opsowl/internal/cases"
	"github.com/reddybashanaguru/opsowl/internal/csvimport"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
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
	casesRepo := cases.NewRepository(q)
	casesSvc := cases.NewService(casesRepo, db, noopEmitter{})
	svc := csvimport.NewService(q, casesSvc, nil)
	h := csvimport.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		csvimport.RegisterRoutes(r, h, jwtMgr)
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

func buildCSVMultipart(t *testing.T, csvContent string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "test.csv")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, _ = io.WriteString(fw, csvContent)
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

// ── Unit tests: SupportedColumns / ColumnMapping ──────────────────────────────

func TestSupportedColumns_NotEmpty(t *testing.T) {
	if len(csvimport.SupportedColumns) == 0 {
		t.Error("SupportedColumns should not be empty")
	}
}

func TestSupportedColumns_ContainsTitle(t *testing.T) {
	found := false
	for _, c := range csvimport.SupportedColumns {
		if c == "title" {
			found = true
		}
	}
	if !found {
		t.Error("SupportedColumns should contain 'title'")
	}
}

// ── GET /imports/columns ──────────────────────────────────────────────────────

func TestListColumns_200(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr@csv.test", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/imports/columns", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestListColumns_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/imports/columns", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestListColumns_403_Analyst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst@csv.test", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst@csv.test", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/imports/columns", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

// ── POST /imports/preview ─────────────────────────────────────────────────────

func TestPreview_200_ValidCSV(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr2@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr2@csv.test", dbgen.UserRoleManager)

	csvData := "title,description,priority\nFraud Alert,Suspicious transaction detected,high\nKYC Breach,Customer KYC expired,medium\n"
	buf, ct := buildCSVMultipart(t, csvData)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", buf)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", ct)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestPreview_400_NoFile(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr3@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr3@csv.test", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Error("expected error status for missing file, got 200")
	}
}

func TestPreview_400_EmptyCSV(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr4@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr4@csv.test", dbgen.UserRoleManager)

	buf, ct := buildCSVMultipart(t, "title,description\n") // header only, no data rows
	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", buf)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", ct)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty CSV, got %d: %s", rr.Code, rr.Body)
	}
}

// ── GET /imports ───────────────────────────────────────────────────────────────

func TestListJobs_200(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr5@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr5@csv.test", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/imports/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
}

// ── GET /imports/{id} — unknown ID ────────────────────────────────────────────

func TestGetJob_404(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr6@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr6@csv.test", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/imports/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 got %d", rr.Code)
	}
}

// ── CSV auto-detect mapping (via Preview) ─────────────────────────────────────

func TestPreview_AutoDetectsMapping(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr7@csv.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr7@csv.test", dbgen.UserRoleManager)

	// Use headers that should auto-map to known fields
	csvData := "Alert Title,Case Description,Severity\nUPI Fraud,Suspect transaction,high\n"
	buf, ct := buildCSVMultipart(t, csvData)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", buf)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", ct)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rr.Code, rr.Body)
	}
	var env struct {
		Data struct {
			DetectedMap []struct {
				CSVHeader string `json:"csv_header"`
				CaseField string `json:"case_field"`
			} `json:"detected_map"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// "Alert Title" should map to "title", "Severity" to "priority"
	mapped := map[string]string{}
	for _, m := range env.Data.DetectedMap {
		mapped[m.CSVHeader] = m.CaseField
	}
	if mapped["Alert Title"] != "title" {
		t.Errorf("expected 'Alert Title' → 'title', got %q", mapped["Alert Title"])
	}
	if mapped["Severity"] != "priority" {
		t.Errorf("expected 'Severity' → 'priority', got %q", mapped["Severity"])
	}
}
