package report_test

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
	"go.uber.org/zap"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/report"
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

func newJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager(
		"../../keys/private.pem",
		"../../keys/public.pem",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}
	return mgr
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
		FullName:     string(role) + " user",
		Role:         role,
	})
	if err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}
	return u.ID
}

func signedBearerToken(t *testing.T, jwtMgr *auth.JWTManager, userID uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := jwtMgr.Sign(userID, email, role)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return "Bearer " + tok
}

func seedCase(t *testing.T, db *sql.DB, creatorID uuid.UUID) uuid.UUID {
	t.Helper()
	q := dbgen.New(db)
	caseSvc := cases.NewService(cases.NewRepository(q), db, noopEmitter{})
	claims := &auth.Claims{UserID: creatorID, Email: "mgr@test.com", Role: dbgen.UserRoleManager}
	c, err := caseSvc.Create(context.Background(), claims, cases.CreateCaseRequest{
		Title:    "STR subject investigation",
		Priority: dbgen.CasePriorityHigh,
	})
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return c.ID
}

type noopEmitter struct{}

func (noopEmitter) Emit(_ context.Context, _ timeline.Event) error { return nil }

func newTestRouter(t *testing.T, db *sql.DB, jwtMgr *auth.JWTManager) http.Handler {
	t.Helper()
	q := dbgen.New(db)
	h := report.NewHandler(q, "", zap.NewNop())
	r := chi.NewRouter()
	report.RegisterRoutes(r, h, jwtMgr)
	return r
}

// strBody returns a minimal valid STR request body.
func strBody() []byte {
	body := map[string]any{
		"report_type":      "STR",
		"suspicion_basis":  "Unusually large cash transactions inconsistent with customer profile",
		"evidence_summary": "Three transactions >10L in one week with no business justification",
		"filing_entity": map[string]any{
			"name":               "Test Bank Ltd",
			"reporting_officer":  "Jane Doe",
		},
		"subject": map[string]any{
			"name":         "John Doe",
			"relationship": "customer",
		},
		"transactions": []map[string]any{},
	}
	b, _ := json.Marshal(body)
	return b
}

// ── C2: Tipping-off controls — role-based STR access ─────────────────────────

func TestRegulatoryReport_AnalystForbidden(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWTManager(t)
	mgrID := seedUser(t, db, "mgr@c2test.com", dbgen.UserRoleManager)
	analystID := seedUser(t, db, "analyst@c2test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, mgrID)

	router := newTestRouter(t, db, jwtMgr)

	// Analyst attempts to generate an STR — must be 403
	req := httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/regulatory-report",
		bytes.NewReader(strBody()),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", signedBearerToken(t, jwtMgr, analystID, "analyst@c2test.com", dbgen.UserRoleAnalyst))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for analyst STR access, got %d", rr.Code)
	}
}

func TestRegulatoryReport_ViewerForbidden(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWTManager(t)
	mgrID := seedUser(t, db, "mgr@c2v.com", dbgen.UserRoleManager)
	viewerID := seedUser(t, db, "viewer@c2v.com", dbgen.UserRoleViewer)
	caseID := seedCase(t, db, mgrID)

	router := newTestRouter(t, db, jwtMgr)

	req := httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/regulatory-report",
		bytes.NewReader(strBody()),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", signedBearerToken(t, jwtMgr, viewerID, "viewer@c2v.com", dbgen.UserRoleViewer))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer STR access, got %d", rr.Code)
	}
}

func TestRegulatoryReport_ManagerAllowed(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWTManager(t)
	mgrID := seedUser(t, db, "mgr@c2m.com", dbgen.UserRoleManager)
	caseID := seedCase(t, db, mgrID)

	router := newTestRouter(t, db, jwtMgr)

	req := httptest.NewRequest(http.MethodPost,
		"/cases/"+caseID.String()+"/regulatory-report",
		bytes.NewReader(strBody()),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", signedBearerToken(t, jwtMgr, mgrID, "mgr@c2m.com", dbgen.UserRoleManager))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Manager should get 200; PDF generate route or regulatory report route
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for manager STR access, got %d body: %s", rr.Code, rr.Body.String())
	}
}

func TestRegulatoryReport_UnauthenticatedRejected(t *testing.T) {
	db, teardown := testDB(t)
	defer teardown()
	jwtMgr := newJWTManager(t)

	router := newTestRouter(t, db, jwtMgr)

	req := httptest.NewRequest(http.MethodPost,
		"/cases/"+uuid.New().String()+"/regulatory-report",
		bytes.NewReader(strBody()),
	)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated STR access, got %d", rr.Code)
	}
}
