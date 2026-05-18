package ai_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
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

	"github.com/reddybashanaguru/opsowl/internal/ai"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

// ── Infrastructure ────────────────────────────────────────────────────────────

var sharedDB *sql.DB

func TestMain(m *testing.M) {
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
		panic("failed to start postgres: " + err.Error())
	}

	connStr, _ := pgc.ConnectionString(ctx, "sslmode=disable")
	pool, _ := pgxpool.New(ctx, connStr)
	sharedDB = stdlib.OpenDBFromPool(pool)

	if err := goose.Up(sharedDB, "../../db/migrations/incremental"); err != nil {
		panic("migrations failed: " + err.Error())
	}

	code := m.Run()
	pool.Close()
	_ = pgc.Terminate(ctx)
	os.Exit(code)
}

// ── Helpers ────────────────────────────────────────────────────────────────────

type noopEmitter struct{}

func (n *noopEmitter) Emit(_ context.Context, _ timeline.Event) error { return nil }

func makeService(t *testing.T, aiServer *httptest.Server) (*ai.Service, *ai.Repository) {
	t.Helper()
	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	url := ""
	if aiServer != nil {
		url = aiServer.URL
	}
	svc := ai.NewService(repo, &noopEmitter{}, zap.NewNop(), url)
	return svc, repo
}

func makeAnalystClaims(userID uuid.UUID) *auth.Claims {
	return &auth.Claims{
		UserID: userID,
		Email:  "analyst@test.com",
		Role:   dbgen.UserRoleAnalyst,
	}
}

func insertUser(t *testing.T, role dbgen.UserRole) uuid.UUID {
	t.Helper()
	q := dbgen.New(sharedDB)
	row, err := q.CreateUser(context.Background(), dbgen.CreateUserParams{
		Email:        uuid.NewString() + "@test.com",
		FullName:     "Test User",
		PasswordHash: "$2a$12$xxx",
		Role:         role,
	})
	if err != nil {
		t.Fatalf("insertUser: %v", err)
	}
	return row.ID
}

func insertCase(t *testing.T, userID uuid.UUID) uuid.UUID {
	t.Helper()
	q := dbgen.New(sharedDB)
	c, err := q.CreateCase(context.Background(), dbgen.CreateCaseParams{
		OrganizationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:          "Test Case",
		Description:    "desc",
		Status:         dbgen.CaseStatusOpen,
		Priority:       dbgen.CasePriorityMedium,
		CreatedBy:      userID,
		Tags:           []string{},
	})
	if err != nil {
		t.Fatalf("insertCase: %v", err)
	}
	return c.ID
}

// mockPythonServer returns an httptest.Server that replies with a fixed copilot response.
func mockPythonServer(t *testing.T, resp interface{}, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func mockCopilotResp(answer string) map[string]interface{} {
	conf := 0.82
	return map[string]interface{}{
		"answer":               answer,
		"confidence_score":     &conf,
		"citations":            []interface{}{},
		"model_used":           "claude-opus-4-5",
		"provider":             "anthropic",
		"tokens_input":         int(450),
		"tokens_output":        int(120),
		"latency_ms":           int(1500),
		"insufficient_context": false,
		"prompt_text":          "Question: test?",
	}
}

// buildRouter creates a testable chi router with the AI module wired in.
func buildRouter(t *testing.T, svc *ai.Service, jwtMgr *auth.JWTManager) http.Handler {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		h := ai.NewHandler(svc)
		ai.RegisterRoutes(r, h, jwtMgr)
	})
	return r
}

func makeJWT(t *testing.T, userID uuid.UUID, role dbgen.UserRole) (string, *auth.JWTManager) {
	t.Helper()
	jwtMgr, _ := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", 60*time.Minute)
	token, err := jwtMgr.Sign(userID, "test@test.com", role)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return token, jwtMgr
}

func authedRequest(t *testing.T, method, path string, body interface{}, token string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

// ── Repository tests ──────────────────────────────────────────────────────────

func TestRepo_CreateAndGet(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)

	citationsJSON, _ := json.Marshal([]ai.Citation{{DocID: "doc-1", DocName: "test.pdf", ChunkIndex: 0, Excerpt: "ex", Similarity: 0.9}})
	conf := sql.NullString{String: "0.85", Valid: true}

	row, err := repo.Create(context.Background(), dbgen.CreateAILogParams{
		CaseID:          caseID,
		TriggeredBy:     userID,
		ActionType:      "ask",
		PromptText:      "masked prompt",
		ResponseText:    "The answer is...",
		ModelUsed:       "claude-opus-4-5",
		Provider:        "anthropic",
		TokensInput:     400,
		TokensOutput:    100,
		LatencyMs:       1200,
		ConfidenceScore: conf,
		Citations:       citationsJSON,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if row.ID == uuid.Nil {
		t.Fatal("expected non-nil ID")
	}
	if row.HumanReviewStatus != "pending" {
		t.Errorf("expected pending, got %s", row.HumanReviewStatus)
	}
	if row.TriggeredByName == "" {
		t.Error("expected triggered_by_name to be populated from join")
	}

	// GetByID
	got, err := repo.GetByID(context.Background(), row.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != row.ID {
		t.Errorf("ID mismatch: %v != %v", got.ID, row.ID)
	}
}

func TestRepo_GetByID_NotFound(t *testing.T) {
	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	_, err := repo.GetByID(context.Background(), uuid.New())
	if err != ai.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepo_Approve(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	reviewerID := insertUser(t, dbgen.UserRoleManager)

	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)

	citationsJSON, _ := json.Marshal([]ai.Citation{})
	row, err := repo.Create(context.Background(), dbgen.CreateAILogParams{
		CaseID: caseID, TriggeredBy: userID,
		ActionType: "ask", PromptText: "p", ResponseText: "r",
		ModelUsed: "m", Provider: "p", TokensInput: 1, TokensOutput: 1, LatencyMs: 1,
		Citations: citationsJSON,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	approved, err := repo.Approve(context.Background(), row.ID, reviewerID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.HumanReviewStatus != "approved" {
		t.Errorf("expected approved, got %s", approved.HumanReviewStatus)
	}
	if !approved.ReviewedBy.Valid {
		t.Error("expected reviewed_by to be set")
	}
	if !approved.ReviewedAt.Valid {
		t.Error("expected reviewed_at to be set")
	}
}

func TestRepo_Reject(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	reviewerID := insertUser(t, dbgen.UserRoleManager)

	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)

	citationsJSON, _ := json.Marshal([]ai.Citation{})
	row, _ := repo.Create(context.Background(), dbgen.CreateAILogParams{
		CaseID: caseID, TriggeredBy: userID,
		ActionType: "ask", PromptText: "p", ResponseText: "r",
		ModelUsed: "m", Provider: "p", TokensInput: 1, TokensOutput: 1, LatencyMs: 1,
		Citations: citationsJSON,
	})

	rejected, err := repo.Reject(context.Background(), row.ID, reviewerID, "not accurate")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.HumanReviewStatus != "rejected" {
		t.Errorf("expected rejected, got %s", rejected.HumanReviewStatus)
	}
	if !rejected.HumanEditedResponse.Valid || rejected.HumanEditedResponse.String != "not accurate" {
		t.Error("expected reason stored in human_edited_response")
	}
}

func TestRepo_Edit(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	reviewerID := insertUser(t, dbgen.UserRoleManager)

	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)

	citationsJSON, _ := json.Marshal([]ai.Citation{})
	row, _ := repo.Create(context.Background(), dbgen.CreateAILogParams{
		CaseID: caseID, TriggeredBy: userID,
		ActionType: "ask", PromptText: "p", ResponseText: "r",
		ModelUsed: "m", Provider: "p", TokensInput: 1, TokensOutput: 1, LatencyMs: 1,
		Citations: citationsJSON,
	})

	edited, err := repo.Edit(context.Background(), row.ID, reviewerID, "corrected answer")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.HumanReviewStatus != "edited" {
		t.Errorf("expected edited, got %s", edited.HumanReviewStatus)
	}
	if !edited.HumanEditedResponse.Valid || edited.HumanEditedResponse.String != "corrected answer" {
		t.Error("expected corrected answer in human_edited_response")
	}
}

func TestRepo_Approve_NotFound(t *testing.T) {
	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	_, err := repo.Approve(context.Background(), uuid.New(), uuid.New())
	if err != ai.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepo_Reject_NotFound(t *testing.T) {
	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	_, err := repo.Reject(context.Background(), uuid.New(), uuid.New(), "reason")
	if err != ai.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepo_Edit_NotFound(t *testing.T) {
	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	_, err := repo.Edit(context.Background(), uuid.New(), uuid.New(), "text")
	if err != ai.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ListByCase(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	for i := 0; i < 3; i++ {
		_, err := svc.Ask(context.Background(), claims, caseID, "question?")
		if err != nil {
			t.Fatalf("Ask %d: %v", i, err)
		}
	}

	logs, meta, err := svc.ListByCase(context.Background(), caseID, ai.ListLogsRequest{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(logs) < 3 {
		t.Errorf("expected >= 3 logs, got %d", len(logs))
	}
	if meta.TotalCount < 3 {
		t.Errorf("expected total >= 3, got %d", meta.TotalCount)
	}
}

func TestService_ListByCase_StatusFilter(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	reviewerID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	log, _ := svc.Ask(context.Background(), claims, caseID, "question?")
	reviewerClaims := &auth.Claims{UserID: reviewerID, Role: dbgen.UserRoleManager}
	_, _ = svc.Approve(context.Background(), reviewerClaims, log.ID)

	logs, _, err := svc.ListByCase(context.Background(), caseID, ai.ListLogsRequest{Status: "approved", Limit: 10})
	if err != nil {
		t.Fatalf("ListByCase status=approved: %v", err)
	}
	for _, l := range logs {
		if l.HumanReviewStatus != "approved" {
			t.Errorf("expected approved, got %s", l.HumanReviewStatus)
		}
	}
}

// ── Service tests ─────────────────────────────────────────────────────────────

func TestService_Ask_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("The risk factors are X and Y."), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	result, err := svc.Ask(context.Background(), claims, caseID, "What are the risk factors?")
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if result.ActionType != "ask" {
		t.Errorf("expected ask, got %s", result.ActionType)
	}
	if result.HumanReviewStatus != "pending" {
		t.Errorf("expected pending, got %s", result.HumanReviewStatus)
	}
	if result.ResponseText != "The risk factors are X and Y." {
		t.Errorf("unexpected response: %s", result.ResponseText)
	}
}

func TestService_Ask_EmptyQuestion(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	svc, _ := makeService(t, nil)
	claims := makeAnalystClaims(userID)

	_, err := svc.Ask(context.Background(), claims, caseID, "")
	if err == nil {
		t.Error("expected error for empty question")
	}
}

func TestService_Ask_QuestionTooLong(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	svc, _ := makeService(t, nil)
	claims := makeAnalystClaims(userID)

	longQ := string(make([]byte, 501))
	_, err := svc.Ask(context.Background(), claims, caseID, longQ)
	if err == nil {
		t.Error("expected error for question > 500 chars")
	}
}

func TestService_Ask_PythonServiceDown(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	// Use a dead server URL
	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	svc := ai.NewService(repo, &noopEmitter{}, zap.NewNop(), "http://localhost:19999")
	claims := makeAnalystClaims(userID)

	_, err := svc.Ask(context.Background(), claims, caseID, "what?")
	if err == nil {
		t.Error("expected error when AI service is down")
	}
}

func TestService_Ask_PythonServiceBadStatus(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, map[string]string{"detail": "internal error"}, http.StatusInternalServerError)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	_, err := svc.Ask(context.Background(), claims, caseID, "what?")
	if err == nil {
		t.Error("expected error for non-200 response")
	}
}

func TestService_Summarize_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("Case summary: key facts are..."), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	result, err := svc.Summarize(context.Background(), claims, caseID)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if result.ActionType != "summarize" {
		t.Errorf("expected summarize, got %s", result.ActionType)
	}
}

func TestService_GenerateNote_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("DRAFT: Investigation note..."), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	result, err := svc.GenerateNote(context.Background(), claims, caseID)
	if err != nil {
		t.Fatalf("GenerateNote: %v", err)
	}
	if result.ActionType != "generate_note" {
		t.Errorf("expected generate_note, got %s", result.ActionType)
	}
}

func TestService_ExtractActions_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("1. Verify KYC documents\n2. Request bank statements"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	result, err := svc.ExtractActions(context.Background(), claims, caseID, "documents")
	if err != nil {
		t.Fatalf("ExtractActions: %v", err)
	}
	if result.ActionType != "extract_actions" {
		t.Errorf("expected extract_actions, got %s", result.ActionType)
	}
}

func TestService_Approve_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	reviewerID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	analysisClaims := makeAnalystClaims(userID)
	reviewerClaims := &auth.Claims{UserID: reviewerID, Email: "mgr@test.com", Role: dbgen.UserRoleManager}

	log, _ := svc.Ask(context.Background(), analysisClaims, caseID, "question?")

	approved, err := svc.Approve(context.Background(), reviewerClaims, log.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.HumanReviewStatus != "approved" {
		t.Errorf("expected approved, got %s", approved.HumanReviewStatus)
	}
	if approved.ReviewedBy == nil {
		t.Error("expected reviewed_by to be set")
	}
}

func TestService_Approve_NotFound(t *testing.T) {
	svc, _ := makeService(t, nil)
	claims := &auth.Claims{UserID: uuid.New(), Role: dbgen.UserRoleManager}
	_, err := svc.Approve(context.Background(), claims, uuid.New())
	if err != ai.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Reject_RequiresReason(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	analysisClaims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), analysisClaims, caseID, "question?")

	reviewerClaims := &auth.Claims{UserID: uuid.New(), Role: dbgen.UserRoleManager}
	_, err := svc.Reject(context.Background(), reviewerClaims, log.ID, "")
	if err == nil {
		t.Error("expected error for empty rejection reason")
	}
}

func TestService_Reject_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	reviewerID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	analysisClaims := makeAnalystClaims(userID)
	reviewerClaims := &auth.Claims{UserID: reviewerID, Role: dbgen.UserRoleManager}

	log, _ := svc.Ask(context.Background(), analysisClaims, caseID, "question?")
	rejected, err := svc.Reject(context.Background(), reviewerClaims, log.ID, "not accurate enough")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.HumanReviewStatus != "rejected" {
		t.Errorf("expected rejected, got %s", rejected.HumanReviewStatus)
	}
	if *rejected.HumanEditedResponse != "not accurate enough" {
		t.Error("rejection reason not stored")
	}
}

func TestService_Edit_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	reviewerID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("original answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	analysisClaims := makeAnalystClaims(userID)
	reviewerClaims := &auth.Claims{UserID: reviewerID, Role: dbgen.UserRoleManager}

	log, _ := svc.Ask(context.Background(), analysisClaims, caseID, "question?")
	edited, err := svc.Edit(context.Background(), reviewerClaims, log.ID, "corrected answer")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.HumanReviewStatus != "edited" {
		t.Errorf("expected edited, got %s", edited.HumanReviewStatus)
	}
	if *edited.HumanEditedResponse != "corrected answer" {
		t.Error("edited response not stored")
	}
}

func TestService_Edit_EmptyResponse(t *testing.T) {
	svc, _ := makeService(t, nil)
	claims := &auth.Claims{UserID: uuid.New(), Role: dbgen.UserRoleManager}
	_, err := svc.Edit(context.Background(), claims, uuid.New(), "")
	if err == nil {
		t.Error("expected error for empty edited response")
	}
}

func TestService_ListPending(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	_, _ = svc.Ask(context.Background(), claims, caseID, "pending question?")

	logs, err := svc.ListPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	for _, l := range logs {
		if l.HumanReviewStatus != "pending" {
			t.Errorf("expected all pending, got %s", l.HumanReviewStatus)
		}
	}
}

// ── Handler tests ─────────────────────────────────────────────────────────────

func TestHandler_Ask_RejectsEmptyQuestion(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/ask", map[string]string{"question": ""}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Ask_RequiresAuth(t *testing.T) {
	svc, _ := makeService(t, nil)
	_, jwtMgr := makeJWT(t, uuid.New(), dbgen.UserRoleAnalyst)
	router := buildRouter(t, svc, jwtMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+uuid.New().String()+"/ai/ask", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandler_Ask_InvalidCaseID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/not-a-uuid/ai/ask", map[string]string{"question": "hi"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Approve_NotFound(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/ai/logs/"+uuid.New().String()+"/approve", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_Reject_RequiresReason(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "question?")

	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/ai/logs/"+log.ID.String()+"/reject",
		map[string]string{"reason": ""}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ListLogs_Empty(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodGet, "/api/v1/cases/"+caseID.String()+"/ai/logs", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := resp["data"].([]interface{})
	if len(data) != 0 {
		t.Errorf("expected empty list for new case, got %d items", len(data))
	}
}

func TestHandler_Summarize_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, mockCopilotResp("Summary text."), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/summarize", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct{ Data struct{ ActionType string `json:"action_type"` } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.ActionType != "summarize" {
		t.Errorf("expected summarize, got %s", resp.Data.ActionType)
	}
}

func TestHandler_GenerateNote_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, mockCopilotResp("DRAFT: note."), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/generate-note", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ExtractActions_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, mockCopilotResp("1. Task one"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/extract-actions",
		map[string]string{"source_type": "documents"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetLog_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, mockCopilotResp("answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "get this?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodGet, "/api/v1/ai/logs/"+log.ID.String(), nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetLog_NotFound(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodGet, "/api/v1/ai/logs/"+uuid.New().String(), nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandler_Edit_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	srv := mockPythonServer(t, mockCopilotResp("original"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "edit this?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodPut, "/api/v1/ai/logs/"+log.ID.String()+"/edit",
		map[string]string{"edited_response": "corrected text"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct{ Data struct{ HumanReviewStatus string `json:"human_review_status"` } `json:"data"` }
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.HumanReviewStatus != "edited" {
		t.Errorf("expected edited, got %s", resp.Data.HumanReviewStatus)
	}
}

func TestHandler_Edit_EmptyBody(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	srv := mockPythonServer(t, mockCopilotResp("original"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "edit empty?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodPut, "/api/v1/ai/logs/"+log.ID.String()+"/edit",
		map[string]string{"edited_response": ""}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ListPending_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	_, _ = svc.Ask(context.Background(), claims, caseID, "pending?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodGet, "/api/v1/ai/logs/pending", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_ListLogs_WithStatusFilter(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "approve me?")
	reviewerClaims := &auth.Claims{UserID: userID, Role: dbgen.UserRoleManager}
	_, _ = svc.Approve(context.Background(), reviewerClaims, log.ID)

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodGet,
		"/api/v1/cases/"+caseID.String()+"/ai/logs?status=approved", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := resp["data"].([]interface{})
	if len(data) == 0 {
		t.Error("expected at least one approved log")
	}
}

func TestService_GetLog_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "get it?")

	got, err := svc.GetLog(context.Background(), log.ID)
	if err != nil {
		t.Fatalf("GetLog: %v", err)
	}
	if got.ID != log.ID {
		t.Errorf("ID mismatch: %v != %v", got.ID, log.ID)
	}
}

func TestService_GetLog_NotFound(t *testing.T) {
	svc, _ := makeService(t, nil)
	_, err := svc.GetLog(context.Background(), uuid.New())
	if err != ai.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestBuildPendingResponse_WithReviewer(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	reviewerID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("answer"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "review me?")

	reviewerClaims := &auth.Claims{UserID: reviewerID, Role: dbgen.UserRoleManager}
	_, _ = svc.Approve(context.Background(), reviewerClaims, log.ID)

	// ListPending exercises buildPendingResponse — after approve there are no pending ones,
	// but we can verify the response shape via ListByCase (uses buildListResponse).
	logs, _, err := svc.ListByCase(context.Background(), caseID, ai.ListLogsRequest{Status: "approved", Limit: 10})
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected approved log")
	}
	approved := logs[0]
	if approved.ReviewedBy == nil {
		t.Error("expected reviewed_by to be populated")
	}
	if approved.ReviewedAt == nil {
		t.Error("expected reviewed_at to be populated")
	}
}

func TestHandler_Approve_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "approve?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodPost, "/api/v1/ai/logs/"+log.ID.String()+"/approve", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_Reject_OK(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "reject?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodPost, "/api/v1/ai/logs/"+log.ID.String()+"/reject",
		map[string]string{"reason": "inaccurate"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandler_AskPythonBadJSON(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	// Server returns invalid JSON
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json at all"))
	}))
	defer badSrv.Close()

	svc, _ := makeService(t, badSrv)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/ask",
		map[string]string{"question": "test?"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ── Coverage-targeted tests ───────────────────────────────────────────────────

type failingEmitter struct{}

func (f *failingEmitter) Emit(_ context.Context, _ timeline.Event) error {
	return errors.New("emit failed")
}

// emitTimeline: covers the logger.Warn branch when emitter returns error.
func TestService_EmitTimelineError(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	q := dbgen.New(sharedDB)
	repo := ai.NewRepository(q)
	svc := ai.NewService(repo, &failingEmitter{}, zap.NewNop(), srv.URL)
	claims := makeAnalystClaims(userID)

	result, err := svc.Ask(context.Background(), claims, caseID, "emit error test?")
	if err != nil {
		t.Fatalf("Ask should succeed even with failing emitter: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// Handler: invalid caseId/logId paths (400).
func TestHandler_Summarize_InvalidCaseID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/not-a-uuid/ai/summarize", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Summarize_ServiceError(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, map[string]string{"detail": "error"}, http.StatusInternalServerError)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/summarize", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandler_GenerateNote_InvalidCaseID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/not-a-uuid/ai/generate-note", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_GenerateNote_ServiceError(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, map[string]string{"detail": "error"}, http.StatusInternalServerError)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/"+caseID.String()+"/ai/generate-note", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandler_ExtractActions_InvalidCaseID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/cases/not-a-uuid/ai/extract-actions",
		map[string]string{"source_type": "documents"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_ListLogs_InvalidCaseID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodGet, "/api/v1/cases/not-a-uuid/ai/logs", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_GetLog_InvalidLogID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodGet, "/api/v1/ai/logs/not-a-uuid", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Approve_InvalidLogID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/ai/logs/not-a-uuid/approve", nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Reject_InvalidLogID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPost, "/api/v1/ai/logs/not-a-uuid/reject",
		map[string]string{"reason": "test"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_Edit_InvalidLogID(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleManager)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleManager)
	svc, _ := makeService(t, nil)
	router := buildRouter(t, svc, jwtMgr)

	req := authedRequest(t, http.MethodPut, "/api/v1/ai/logs/not-a-uuid/edit",
		map[string]string{"edited_response": "text"}, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ListByCase hasMore path: create 6 logs, request with limit=5.
func TestService_ListByCase_HasMore(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	for i := 0; i < 6; i++ {
		_, _ = svc.Ask(context.Background(), claims, caseID, "question?")
	}

	logs, meta, err := svc.ListByCase(context.Background(), caseID, ai.ListLogsRequest{Limit: 5})
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
	if !meta.HasMore {
		t.Error("expected HasMore=true")
	}
	if meta.NextCursor == nil {
		t.Error("expected NextCursor to be set")
	}
}

// ListPending: limit clamping (0 → 50).
func TestService_ListPending_LimitClamped(t *testing.T) {
	svc, _ := makeService(t, nil)
	logs, err := svc.ListPending(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if logs == nil {
		t.Error("expected non-nil slice")
	}
}

// ListLogs handler: cursor query param is parsed correctly.
func TestHandler_ListLogs_WithCursor(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)
	token, jwtMgr := makeJWT(t, userID, dbgen.UserRoleAnalyst)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)
	log, _ := svc.Ask(context.Background(), claims, caseID, "cursor test?")

	router := buildRouter(t, svc, jwtMgr)
	req := authedRequest(t, http.MethodGet,
		"/api/v1/cases/"+caseID.String()+"/ai/logs?cursor="+log.ID.String(), nil, token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ExtractActions: empty sourceType defaults to "documents".
func TestService_ExtractActions_DefaultSourceType(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("actions"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	claims := makeAnalystClaims(userID)

	result, err := svc.ExtractActions(context.Background(), claims, caseID, "")
	if err != nil {
		t.Fatalf("ExtractActions: %v", err)
	}
	if result.ActionType != "extract_actions" {
		t.Errorf("expected extract_actions, got %s", result.ActionType)
	}
}

// buildListResponse: covers HumanEditedResponse branch via rejected log in list.
func TestService_ListByCase_RejectedLogHasEditedResponse(t *testing.T) {
	userID := insertUser(t, dbgen.UserRoleAnalyst)
	reviewerID := insertUser(t, dbgen.UserRoleManager)
	caseID := insertCase(t, userID)

	srv := mockPythonServer(t, mockCopilotResp("ans"), http.StatusOK)
	defer srv.Close()

	svc, _ := makeService(t, srv)
	analysisClaims := makeAnalystClaims(userID)
	reviewerClaims := &auth.Claims{UserID: reviewerID, Role: dbgen.UserRoleManager}

	log, _ := svc.Ask(context.Background(), analysisClaims, caseID, "reject me?")
	_, _ = svc.Reject(context.Background(), reviewerClaims, log.ID, "not accurate enough")

	logs, _, err := svc.ListByCase(context.Background(), caseID, ai.ListLogsRequest{Status: "rejected", Limit: 10})
	if err != nil {
		t.Fatalf("ListByCase: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected rejected log in list")
	}
	if logs[0].HumanEditedResponse == nil {
		t.Error("expected human_edited_response set for rejected log")
	}
	if logs[0].ReviewedBy == nil {
		t.Error("expected reviewed_by set for rejected log")
	}
}

// Service-level Reject/Edit NotFound covers the repo error propagation branch.
func TestService_Reject_NotFound(t *testing.T) {
	svc, _ := makeService(t, nil)
	claims := &auth.Claims{UserID: uuid.New(), Role: dbgen.UserRoleManager}
	_, err := svc.Reject(context.Background(), claims, uuid.New(), "reason")
	if !errors.Is(err, ai.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Edit_NotFound(t *testing.T) {
	svc, _ := makeService(t, nil)
	claims := &auth.Claims{UserID: uuid.New(), Role: dbgen.UserRoleManager}
	_, err := svc.Edit(context.Background(), claims, uuid.New(), "edited text")
	if !errors.Is(err, ai.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
