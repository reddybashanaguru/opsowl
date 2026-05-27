package scheduled_reports_test

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
	scheduled_reports "github.com/reddybashanaguru/opsowl/internal/scheduled_reports"
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
	svc := scheduled_reports.NewService(q, nil, nil) // nil mailer/logger — fine for CRUD tests
	h := scheduled_reports.NewHandler(svc)
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		scheduled_reports.RegisterRoutes(r, h, jwtMgr)
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
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

// ── Unit: CalcNextRun ──────────────────────────────────────────────────────────

func TestCalcNextRun_Daily_FutureHour(t *testing.T) {
	// Use hour 23 UTC — should always be today or tomorrow, never in the past.
	next := scheduled_reports.CalcNextRun(scheduled_reports.FreqDaily, 23, nil)
	if !next.After(time.Now().UTC()) {
		t.Errorf("CalcNextRun should return a future time, got %v", next)
	}
	if next.Hour() != 23 {
		t.Errorf("expected hour 23, got %d", next.Hour())
	}
}

func TestCalcNextRun_Daily_PastHour_AdvancesToTomorrow(t *testing.T) {
	// Hour 0 UTC: if it's already past midnight, next run is tomorrow 00:00.
	next := scheduled_reports.CalcNextRun(scheduled_reports.FreqDaily, 0, nil)
	if !next.After(time.Now().UTC()) {
		t.Errorf("CalcNextRun(hour=0) should be in the future, got %v", next)
	}
}

func TestCalcNextRun_Weekly_CorrectDayOfWeek(t *testing.T) {
	dow := int32(1) // Monday
	next := scheduled_reports.CalcNextRun(scheduled_reports.FreqWeekly, 9, &dow)
	if next.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", next.Weekday())
	}
	if !next.After(time.Now().UTC()) {
		t.Errorf("next weekly run should be in the future, got %v", next)
	}
}

func TestCalcNextRun_Weekly_Sunday(t *testing.T) {
	dow := int32(0) // Sunday
	next := scheduled_reports.CalcNextRun(scheduled_reports.FreqWeekly, 8, &dow)
	if next.Weekday() != time.Sunday {
		t.Errorf("expected Sunday, got %v", next.Weekday())
	}
}

// ── Unit: EmailsFromString ────────────────────────────────────────────────────

func TestEmailsFromString_CommaSeparated(t *testing.T) {
	got := scheduled_reports.EmailsFromString("a@example.com, b@example.com , c@example.com")
	if len(got) != 3 {
		t.Errorf("expected 3 emails, got %d: %v", len(got), got)
	}
}

func TestEmailsFromString_Empty(t *testing.T) {
	got := scheduled_reports.EmailsFromString("")
	if len(got) != 0 {
		t.Errorf("expected 0 emails for empty string, got %d", len(got))
	}
}

func TestEmailsFromString_SingleEmail(t *testing.T) {
	got := scheduled_reports.EmailsFromString("ops@example.com")
	if len(got) != 1 || got[0] != "ops@example.com" {
		t.Errorf("expected single email, got %v", got)
	}
}

func TestEmailsFromString_TrimsWhitespace(t *testing.T) {
	got := scheduled_reports.EmailsFromString("  a@x.com  ,  b@x.com  ")
	for _, e := range got {
		if e != "a@x.com" && e != "b@x.com" {
			t.Errorf("unexpected email value %q after trim", e)
		}
	}
}

// ── RBAC ──────────────────────────────────────────────────────────────────────

func TestList_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report-schedules/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestList_403_Analyst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	aID := seedUser(t, db, "analyst@sr.test", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, aID, "analyst@sr.test", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report-schedules/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestList_200_Manager(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr@sr.test", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report-schedules/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body)
	}
}

// ── CRUD lifecycle ────────────────────────────────────────────────────────────

func createSchedule(t *testing.T, r http.Handler, tok string) string {
	t.Helper()
	hour := int32(8)
	body := jsonBody(t, map[string]any{
		"name":       "Daily Summary",
		"frequency":  "daily",
		"recipients": []string{"ops@example.com"},
		"send_hour":  &hour,
		"is_active":  true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report-schedules/", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("createSchedule: expected 201 got %d: %s", rr.Code, rr.Body)
	}
	var env struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	return env.Data.ID
}

func TestCreate_201(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr2@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr2@sr.test", dbgen.UserRoleManager)

	id := createSchedule(t, r, tok)
	if id == "" {
		t.Error("expected non-empty schedule ID")
	}
}

func TestCreate_400_MissingRecipients(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr3@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr3@sr.test", dbgen.UserRoleManager)

	hour := int32(8)
	body := jsonBody(t, map[string]any{
		"name": "No Recipients", "frequency": "daily",
		"recipients": []string{}, "send_hour": &hour,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report-schedules/", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing recipients, got %d", rr.Code)
	}
}

func TestCreate_400_InvalidFrequency(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr4@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr4@sr.test", dbgen.UserRoleManager)

	hour := int32(8)
	body := jsonBody(t, map[string]any{
		"name": "Bad Freq", "frequency": "monthly",
		"recipients": []string{"a@b.com"}, "send_hour": &hour,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report-schedules/", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid frequency, got %d", rr.Code)
	}
}

func TestGetUpdateDelete_Lifecycle(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr5@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr5@sr.test", dbgen.UserRoleManager)

	id := createSchedule(t, r, tok)

	// GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/report-schedules/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d", rr.Code)
	}

	// PUT update
	hour := int32(10)
	body := jsonBody(t, map[string]any{
		"name": "Updated Summary", "frequency": "daily",
		"recipients": []string{"updated@example.com"}, "send_hour": &hour, "is_active": false,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/report-schedules/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT: expected 200, got %d: %s", rr.Code, rr.Body)
	}

	// DELETE
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/report-schedules/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("DELETE: expected 204, got %d", rr.Code)
	}

	// GET after delete → 404
	req = httptest.NewRequest(http.MethodGet, "/api/v1/report-schedules/"+id, nil)
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
	mgrID := seedUser(t, db, "mgr6@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr6@sr.test", dbgen.UserRoleManager)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report-schedules/"+uuid.New().String(), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestCreate_400_WeeklyMissingDOW(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	mgrID := seedUser(t, db, "mgr7@sr.test", dbgen.UserRoleManager)
	tok := makeToken(t, jwtMgr, mgrID, "mgr7@sr.test", dbgen.UserRoleManager)

	hour := int32(9)
	body := jsonBody(t, map[string]any{
		"name": "Weekly No DOW", "frequency": "weekly",
		"recipients": []string{"a@b.com"}, "send_hour": &hour,
		// day_of_week intentionally omitted
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report-schedules/", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for weekly schedule without day_of_week, got %d: %s", rr.Code, rr.Body)
	}
}
