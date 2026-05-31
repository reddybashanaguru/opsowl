package insights_test

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
	"github.com/reddybashanaguru/opsowl/internal/insights"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
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

func seedCase(t *testing.T, db *sql.DB, creatorID uuid.UUID) uuid.UUID {
	t.Helper()
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	var caseID uuid.UUID
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO cases (case_number, title, description, status, priority, created_by, organization_id, tags)
		 VALUES ('OWL-' || nextval('case_number_seq'), 'Test Case', '', 'open', 'high', $1, $2, '{}')
		 RETURNING id`,
		creatorID, orgID,
	).Scan(&caseID)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return caseID
}

func setupRouter(t *testing.T, db *sql.DB, aiURL string) (*chi.Mux, *auth.JWTManager) {
	t.Helper()

	jwtMgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}

	q := dbgen.New(db)
	h := insights.NewHandler(q, aiURL, timeline.NoopEmitter{})

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		insights.RegisterRoutes(r, h, jwtMgr)
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

// mockAIServer returns a test server that responds with a canned AI analyse response.
func mockAIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"risk_score":          72.5,
			"risk_factors":        []string{"unusual transfer pattern"},
			"entities":            map[string]interface{}{"individuals": []string{}, "companies": []string{}},
			"anomalies":           []string{"multiple accounts"},
			"recommended_actions": []string{"freeze account"},
			"model_used":          "mock-model",
			"confidence":          0.85,
		})
	}))
}

// ── GET /cases/:id/insights ───────────────────────────────────────────────────

func TestGetInsights_NoAnalysis(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := mockAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst@ins.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst@ins.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID.String()+"/insights", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body struct {
		Data struct {
			AIAvailable bool   `json:"ai_available"`
			RiskLevel   string `json:"risk_level"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if body.Data.AIAvailable {
		t.Error("expected ai_available=false when no analysis exists")
	}
	if body.Data.RiskLevel != "unknown" {
		t.Errorf("expected risk_level=unknown, got %s", body.Data.RiskLevel)
	}
}

func TestGetInsights_401_MissingToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := mockAIServer(t)
	defer ai.Close()

	r, _ := setupRouter(t, db, ai.URL)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+uuid.New().String()+"/insights", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestGetInsights_400_InvalidID(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := mockAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst2@ins.com", dbgen.UserRoleAnalyst)
	tok := makeToken(t, jwtMgr, analystID, "analyst2@ins.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/not-a-uuid/insights", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ── POST /cases/:id/insights/analyse ─────────────────────────────────────────

func TestAnalyseInsights_StoresResult(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := mockAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst3@ins.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst3@ins.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/insights/analyse",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Analyse now returns 202 Accepted — work is dispatched to the background pool.
	// In tests worker.Background is nil so worker.TrySubmit runs synchronously,
	// meaning the insights are stored before the handler returns.
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 accepted, got %d: %s", rr.Code, rr.Body.String())
	}
	var queued struct {
		Data struct {
			Queued bool   `json:"queued"`
			CaseID string `json:"case_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &queued); err != nil {
		t.Fatalf("parse queued body: %v", err)
	}
	if !queued.Data.Queued {
		t.Error("expected queued=true in response")
	}

	// Verify stored result via GET insights (sync in test context — no polling needed).
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID.String()+"/insights", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	var body struct {
		Data struct {
			AIAvailable bool    `json:"ai_available"`
			RiskLevel   string  `json:"risk_level"`
			RiskScore   float64 `json:"risk_score"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse insights body: %v", err)
	}
	if !body.Data.AIAvailable {
		t.Error("expected ai_available=true after analysis")
	}
	if body.Data.RiskLevel != "medium" {
		t.Errorf("expected risk_level=medium (score=72.5 is <75), got %s", body.Data.RiskLevel)
	}
	if body.Data.RiskScore < 70 || body.Data.RiskScore > 75 {
		t.Errorf("expected risk_score ~72.5, got %v", body.Data.RiskScore)
	}
}

func TestAnalyseInsights_AIUnavailable_ReturnsEmpty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	// AI server that always fails
	badAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer badAI.Close()

	r, jwtMgr := setupRouter(t, db, badAI.URL)
	analystID := seedUser(t, db, "analyst4@ins.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst4@ins.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/insights/analyse",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Analyse always returns 202 immediately regardless of AI availability.
	// The background worker silently drops the job when AI fails.
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 accepted, got %d: %s", rr.Code, rr.Body.String())
	}

	// GET insights should still return ai_available=false (no result was stored).
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID.String()+"/insights", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	var body struct {
		Data struct{ AIAvailable bool `json:"ai_available"` } `json:"data"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &body)
	if body.Data.AIAvailable {
		t.Error("expected ai_available=false when AI service is down")
	}
}

func TestAnalyseInsights_403_ViewerForbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := mockAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	viewerID := seedUser(t, db, "viewer@ins.com", dbgen.UserRoleViewer)
	caseID := seedCase(t, db, viewerID)
	tok := makeToken(t, jwtMgr, viewerID, "viewer@ins.com", dbgen.UserRoleViewer)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/insights/analyse",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer role, got %d", rr.Code)
	}
}
