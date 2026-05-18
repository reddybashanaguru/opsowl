package comms_test

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
	"github.com/reddybashanaguru/opsowl/internal/comms"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
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

// noopAIServer accepts any request and returns an empty valid response.
func noopAIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"entities":     map[string]interface{}{},
			"action_items": []string{},
			"sentiment":    "neutral",
			"model_used":   "mock",
		})
	}))
}

func setupRouter(t *testing.T, db *sql.DB, aiURL string) (*chi.Mux, *auth.JWTManager) {
	t.Helper()

	jwtMgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}

	q := dbgen.New(db)
	logger := zap.NewNop()
	svc := comms.NewService(q, aiURL, logger)
	h := comms.NewHandler(svc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		comms.RegisterRoutes(r, h, jwtMgr)
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

// ── GET /cases/:caseId/comms ──────────────────────────────────────────────────

func TestListComms_Empty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := noopAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst@comms.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst@comms.com", dbgen.UserRoleAnalyst)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID.String()+"/comms", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body struct {
		Data []interface{} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(body.Data) != 0 {
		t.Errorf("expected 0 comms, got %d", len(body.Data))
	}
}

func TestListComms_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := noopAIServer(t)
	defer ai.Close()

	r, _ := setupRouter(t, db, ai.URL)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+uuid.New().String()+"/comms", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── POST /cases/:caseId/comms ─────────────────────────────────────────────────

func TestCreateComm_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := noopAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst2@comms.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst2@comms.com", dbgen.UserRoleAnalyst)

	body := map[string]string{
		"channel":      "email",
		"direction":    "outbound",
		"contact_name": "John Doe",
		"body":         "Requesting account statement for review.",
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/comms",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data struct {
			ID          string `json:"id"`
			Channel     string `json:"channel"`
			Direction   string `json:"direction"`
			ContactName string `json:"contact_name"`
			Body        string `json:"body"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.Data.Channel != "email" {
		t.Errorf("expected channel=email, got %s", resp.Data.Channel)
	}
	if resp.Data.Direction != "outbound" {
		t.Errorf("expected direction=outbound, got %s", resp.Data.Direction)
	}
	if resp.Data.ContactName != "John Doe" {
		t.Errorf("expected contact_name=John Doe, got %s", resp.Data.ContactName)
	}
	if resp.Data.ID == "" {
		t.Error("expected non-empty id")
	}
}

func TestCreateComm_400_MissingFields(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := noopAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst3@comms.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst3@comms.com", dbgen.UserRoleAnalyst)

	// Missing body and contact_name
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/comms",
		bytes.NewBufferString(`{"channel":"email","direction":"inbound"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// ── DELETE /cases/:caseId/comms/:commId ───────────────────────────────────────

func TestDeleteComm_Success(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := noopAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	analystID := seedUser(t, db, "analyst4@comms.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, analystID)
	tok := makeToken(t, jwtMgr, analystID, "analyst4@comms.com", dbgen.UserRoleAnalyst)

	// First create a comm log
	bodyJSON, _ := json.Marshal(map[string]string{
		"channel":      "call",
		"direction":    "inbound",
		"contact_name": "Jane Smith",
		"body":         "Borrower called to dispute the case.",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/comms",
		bytes.NewReader(bodyJSON))
	createReq.Header.Set("Authorization", "Bearer "+tok)
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	r.ServeHTTP(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("create comm failed: %d %s", createRR.Code, createRR.Body.String())
	}

	var created struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	_ = json.Unmarshal(createRR.Body.Bytes(), &created)
	commID := created.Data.ID

	// Now delete it
	delReq := httptest.NewRequest(http.MethodDelete,
		"/api/v1/cases/"+caseID.String()+"/comms/"+commID, nil)
	delReq.Header.Set("Authorization", "Bearer "+tok)
	delRR := httptest.NewRecorder()
	r.ServeHTTP(delRR, delReq)

	if delRR.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", delRR.Code, delRR.Body.String())
	}
}

func TestCreateComm_403_ViewerForbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	ai := noopAIServer(t)
	defer ai.Close()

	r, jwtMgr := setupRouter(t, db, ai.URL)
	viewerID := seedUser(t, db, "viewer@comms.com", dbgen.UserRoleViewer)
	caseID := seedCase(t, db, viewerID)
	tok := makeToken(t, jwtMgr, viewerID, "viewer@comms.com", dbgen.UserRoleViewer)

	bodyJSON, _ := json.Marshal(map[string]string{
		"channel":      "email",
		"direction":    "outbound",
		"contact_name": "Someone",
		"body":         "test",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID.String()+"/comms",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer, got %d", rr.Code)
	}
}
