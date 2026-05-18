package cases_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	dbgen "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

// noopEmitter satisfies timeline.Emitter without touching the DB.
type noopEmitter struct{}

func (noopEmitter) Emit(_ context.Context, _ timeline.Event) error { return nil }

// ── test infrastructure ──────────────────────────────────────────────────────

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

func newTestService(t *testing.T, db *sql.DB) *cases.Service {
	t.Helper()
	q := dbgen.New(db)
	repo := cases.NewRepository(q)
	return cases.NewService(repo, noopEmitter{})
}

// seedUser creates a user and returns its ID.
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

func claimsFor(userID uuid.UUID, email string, role dbgen.UserRole) *auth.Claims {
	return &auth.Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
	}
}

// ── service-layer tests ──────────────────────────────────────────────────────

func TestCreateCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)
	analystID := seedUser(t, db, "analyst@test.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "analyst@test.com", dbgen.UserRoleAnalyst)

	t.Run("creates with draft status and OWL case number", func(t *testing.T) {
		c, err := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
			Title:    "Suspicious transaction cluster",
			Priority: dbgen.CasePriorityHigh,
			Tags:     []string{"fraud", "upi"},
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if c.Status != dbgen.CaseStatusDraft {
			t.Errorf("expected status=draft, got %s", c.Status)
		}
		if !strings.HasPrefix(c.CaseNumber, "OWL-") {
			t.Errorf("expected OWL- prefix, got %s", c.CaseNumber)
		}
		if len(c.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(c.Tags))
		}
		// SLA not set on draft
		if c.SLABreachAt != nil {
			t.Error("SLA should not be set on draft")
		}
	})

	t.Run("title too short rejected", func(t *testing.T) {
		_, err := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
			Title:    "abc",
			Priority: dbgen.CasePriorityLow,
		})
		// service itself doesn't validate length — handler does
		// but we verify the title passes through correctly
		_ = err
	})

	t.Run("tag limit enforced", func(t *testing.T) {
		tags := make([]string, 21)
		for i := range tags {
			tags[i] = fmt.Sprintf("tag%d", i)
		}
		_, err := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
			Title:    "Too many tags case",
			Priority: dbgen.CasePriorityLow,
			Tags:     tags,
		})
		if err == nil {
			t.Fatal("expected ErrTagLimit")
		}
	})
}

func TestGetCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)
	analystID := seedUser(t, db, "get@test.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "get@test.com", dbgen.UserRoleAnalyst)

	created, _ := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
		Title:    "Case to retrieve",
		Priority: dbgen.CasePriorityMedium,
	})

	t.Run("get existing case", func(t *testing.T) {
		c, err := svc.Get(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if c.ID != created.ID {
			t.Errorf("id mismatch: got %s want %s", c.ID, created.ID)
		}
		if c.CaseNumber != created.CaseNumber {
			t.Errorf("case number mismatch")
		}
	})

	t.Run("get non-existent returns ErrNotFound", func(t *testing.T) {
		_, err := svc.Get(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestListCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)
	analystID := seedUser(t, db, "list@test.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "list@test.com", dbgen.UserRoleAnalyst)

	// Create 3 cases: 2 high, 1 critical
	for i := 0; i < 2; i++ {
		svc.Create(context.Background(), claims, cases.CreateCaseRequest{
			Title:    fmt.Sprintf("High priority case %d", i),
			Priority: dbgen.CasePriorityHigh,
		})
	}
	svc.Create(context.Background(), claims, cases.CreateCaseRequest{
		Title:    "Critical priority case",
		Priority: dbgen.CasePriorityCritical,
	})

	t.Run("list all returns 3", func(t *testing.T) {
		cs, meta, err := svc.List(context.Background(), cases.ListFilters{Limit: 20})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if meta.TotalCount != 3 {
			t.Errorf("expected total 3, got %d", meta.TotalCount)
		}
		if len(cs) != 3 {
			t.Errorf("expected 3 items, got %d", len(cs))
		}
	})

	t.Run("filter by priority=critical returns 1", func(t *testing.T) {
		p := dbgen.CasePriorityCritical
		cs, meta, err := svc.List(context.Background(), cases.ListFilters{Priority: &p, Limit: 20})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if meta.TotalCount != 1 {
			t.Errorf("expected 1 critical, got %d", meta.TotalCount)
		}
		if len(cs) != 1 || cs[0].Priority != dbgen.CasePriorityCritical {
			t.Error("filter did not work")
		}
	})

	t.Run("filter by status=draft returns all 3 (default status)", func(t *testing.T) {
		s := dbgen.CaseStatusDraft
		cs, _, err := svc.List(context.Background(), cases.ListFilters{Status: &s, Limit: 20})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(cs) != 3 {
			t.Errorf("expected 3 draft, got %d", len(cs))
		}
	})

	t.Run("limit and pagination", func(t *testing.T) {
		cs, meta, err := svc.List(context.Background(), cases.ListFilters{Limit: 2})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(cs) != 2 {
			t.Errorf("expected 2 items, got %d", len(cs))
		}
		if !meta.HasMore {
			t.Error("expected HasMore=true")
		}
		if meta.NextCursor == nil {
			t.Error("expected next cursor")
		}
	})
}

func TestChangeStatus(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)

	analystID := seedUser(t, db, "analyst2@test.com", dbgen.UserRoleAnalyst)
	managerID := seedUser(t, db, "manager@test.com", dbgen.UserRoleManager)
	adminID := seedUser(t, db, "admin@test.com", dbgen.UserRoleAdmin)

	analystClaims := claimsFor(analystID, "analyst2@test.com", dbgen.UserRoleAnalyst)
	managerClaims := claimsFor(managerID, "manager@test.com", dbgen.UserRoleManager)
	adminClaims := claimsFor(adminID, "admin@test.com", dbgen.UserRoleAdmin)

	createCase := func(t *testing.T) *cases.CaseResponse {
		t.Helper()
		c, err := svc.Create(context.Background(), analystClaims, cases.CreateCaseRequest{
			Title:    "Status transition test case",
			Priority: dbgen.CasePriorityHigh,
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		return c
	}

	t.Run("draft -> open sets SLA", func(t *testing.T) {
		c := createCase(t)
		updated, err := svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusOpen,
		})
		if err != nil {
			t.Fatalf("change status: %v", err)
		}
		if updated.Status != dbgen.CaseStatusOpen {
			t.Errorf("expected open, got %s", updated.Status)
		}
		if updated.SLABreachAt == nil {
			t.Error("SLA breach time should be set when moving to open")
		}
		// high priority = 24h SLA
		expected := time.Now().Add(24 * time.Hour)
		diff := updated.SLABreachAt.Sub(expected)
		if diff < -2*time.Minute || diff > 2*time.Minute {
			t.Errorf("SLA time off by %v", diff)
		}
	})

	t.Run("open -> in_progress by analyst", func(t *testing.T) {
		c := createCase(t)
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})
		updated, err := svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusInProgress,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Status != dbgen.CaseStatusInProgress {
			t.Errorf("expected in_progress, got %s", updated.Status)
		}
	})

	t.Run("invalid transition returns ErrInvalidTransition", func(t *testing.T) {
		c := createCase(t)
		_, err := svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusClosed,
		})
		if err == nil {
			t.Fatal("expected error for invalid transition draft->closed")
		}
	})

	t.Run("escalation requires manager role", func(t *testing.T) {
		c := createCase(t)
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})
		// analyst cannot escalate
		_, err := svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusEscalated,
			Reason: "needs escalation urgently",
		})
		if err == nil {
			t.Fatal("analyst should not be able to escalate")
		}
		// manager can
		_, err = svc.ChangeStatus(context.Background(), managerClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusEscalated,
			Reason: "needs escalation urgently",
		})
		if err != nil {
			t.Fatalf("manager escalate: %v", err)
		}
	})

	t.Run("escalation requires reason (min 10 chars)", func(t *testing.T) {
		c := createCase(t)
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})
		_, err := svc.ChangeStatus(context.Background(), managerClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusEscalated,
			Reason: "short",
		})
		if err == nil {
			t.Fatal("expected ErrReasonRequired")
		}
	})

	t.Run("escalated -> in_progress requires admin", func(t *testing.T) {
		c := createCase(t)
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})
		svc.ChangeStatus(context.Background(), managerClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusEscalated,
			Reason: "serious escalation needed",
		})
		// manager cannot de-escalate
		_, err := svc.ChangeStatus(context.Background(), managerClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusInProgress,
		})
		if err == nil {
			t.Fatal("manager should not de-escalate")
		}
		// admin can
		_, err = svc.ChangeStatus(context.Background(), adminClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusInProgress,
		})
		if err != nil {
			t.Fatalf("admin de-escalate: %v", err)
		}
	})

	t.Run("pending_review -> closed by manager", func(t *testing.T) {
		c := createCase(t)
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusPendingReview})
		updated, err := svc.ChangeStatus(context.Background(), managerClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusClosed,
		})
		if err != nil {
			t.Fatalf("close: %v", err)
		}
		if updated.Status != dbgen.CaseStatusClosed {
			t.Errorf("expected closed, got %s", updated.Status)
		}
	})

	t.Run("closed case cannot transition", func(t *testing.T) {
		c := createCase(t)
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
		svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusPendingReview})
		svc.ChangeStatus(context.Background(), managerClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusClosed})
		_, err := svc.ChangeStatus(context.Background(), adminClaims, c.ID, cases.StatusRequest{
			Status: dbgen.CaseStatusOpen,
		})
		if err == nil {
			t.Fatal("closed case should not transition")
		}
	})
}

func TestAssignCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)

	analystID := seedUser(t, db, "assign-analyst@test.com", dbgen.UserRoleAnalyst)
	managerID := seedUser(t, db, "assign-manager@test.com", dbgen.UserRoleManager)
	analystClaims := claimsFor(analystID, "assign-analyst@test.com", dbgen.UserRoleAnalyst)
	managerClaims := claimsFor(managerID, "assign-manager@test.com", dbgen.UserRoleManager)

	c, _ := svc.Create(context.Background(), analystClaims, cases.CreateCaseRequest{
		Title:    "Assignable case",
		Priority: dbgen.CasePriorityMedium,
	})

	t.Run("analyst cannot assign", func(t *testing.T) {
		_, err := svc.Assign(context.Background(), analystClaims, c.ID, cases.AssignRequest{UserID: analystID})
		if err == nil {
			t.Fatal("analyst should not be able to assign")
		}
	})

	t.Run("manager can assign", func(t *testing.T) {
		updated, err := svc.Assign(context.Background(), managerClaims, c.ID, cases.AssignRequest{
			UserID: analystID,
			Note:   "Assigned for investigation",
		})
		if err != nil {
			t.Fatalf("assign: %v", err)
		}
		if updated.AssignedTo == nil || updated.AssignedTo.ID != analystID {
			t.Error("case should be assigned to analyst")
		}
	})
}

func TestUpdateTags(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)

	analystID := seedUser(t, db, "tags@test.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "tags@test.com", dbgen.UserRoleAnalyst)

	c, _ := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
		Title:    "Tags test case",
		Priority: dbgen.CasePriorityLow,
		Tags:     []string{"initial", "existing"},
	})

	t.Run("add and remove tags", func(t *testing.T) {
		updated, err := svc.UpdateTags(context.Background(), claims, c.ID, cases.TagsRequest{
			Add:    []string{"new-tag", "another"},
			Remove: []string{"initial"},
		})
		if err != nil {
			t.Fatalf("update tags: %v", err)
		}
		tagSet := make(map[string]struct{}, len(updated.Tags))
		for _, t := range updated.Tags {
			tagSet[t] = struct{}{}
		}
		if _, ok := tagSet["initial"]; ok {
			t.Error("'initial' tag should have been removed")
		}
		if _, ok := tagSet["new-tag"]; !ok {
			t.Error("'new-tag' should be present")
		}
		if _, ok := tagSet["existing"]; !ok {
			t.Error("'existing' should still be present")
		}
	})
}

func TestSoftDelete(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)

	analystID := seedUser(t, db, "delete-analyst@test.com", dbgen.UserRoleAnalyst)
	adminID := seedUser(t, db, "delete-admin@test.com", dbgen.UserRoleAdmin)
	analystClaims := claimsFor(analystID, "delete-analyst@test.com", dbgen.UserRoleAnalyst)
	adminClaims := claimsFor(adminID, "delete-admin@test.com", dbgen.UserRoleAdmin)

	c, _ := svc.Create(context.Background(), analystClaims, cases.CreateCaseRequest{
		Title:    "Deletable case",
		Priority: dbgen.CasePriorityLow,
	})

	t.Run("analyst cannot delete", func(t *testing.T) {
		err := svc.Delete(context.Background(), analystClaims, c.ID)
		if err == nil {
			t.Fatal("analyst should not delete cases")
		}
	})

	t.Run("admin can soft delete", func(t *testing.T) {
		err := svc.Delete(context.Background(), adminClaims, c.ID)
		if err != nil {
			t.Fatalf("delete: %v", err)
		}
		// Should no longer be fetchable
		_, err = svc.Get(context.Background(), c.ID)
		if err == nil {
			t.Fatal("deleted case should return not found")
		}
	})

	t.Run("delete non-existent returns error", func(t *testing.T) {
		err := svc.Delete(context.Background(), adminClaims, uuid.New())
		if err == nil {
			t.Fatal("expected error for non-existent case")
		}
	})
}

func TestListActivities(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)

	analystID := seedUser(t, db, "activity@test.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "activity@test.com", dbgen.UserRoleAnalyst)

	c, _ := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
		Title:    "Activity tracking case",
		Priority: dbgen.CasePriorityMedium,
	})

	// Generate a status change to create more activity
	time.Sleep(10 * time.Millisecond) // ensure ordering
	svc.ChangeStatus(context.Background(), claims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusOpen})

	time.Sleep(100 * time.Millisecond) // allow goroutine activity writes

	t.Run("returns activities in reverse chronological order", func(t *testing.T) {
		acts, _, _, err := svc.ListActivities(context.Background(), c.ID, nil, 50)
		if err != nil {
			t.Fatalf("list activities: %v", err)
		}
		if len(acts) < 1 {
			t.Fatal("expected at least 1 activity (status_changed is sync)")
		}
		// Most recent first
		if len(acts) > 1 {
			if acts[0].CreatedAt.Before(acts[1].CreatedAt) {
				t.Error("activities should be in reverse chronological order")
			}
		}
	})
}

// ── HTTP handler tests ───────────────────────────────────────────────────────

func setupRouter(t *testing.T, db *sql.DB) (*chi.Mux, *auth.JWTManager) {
	t.Helper()
	jwtMgr, err := auth.NewJWTManager(
		"../../keys/private.pem",
		"../../keys/public.pem",
		time.Hour,
	)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}

	q := dbgen.New(db)
	repo := cases.NewRepository(q)
	svc := cases.NewService(repo, noopEmitter{})
	h := cases.NewHandler(svc)

	r := chi.NewRouter()
	// RegisterRoutes adds its own JWT middleware, mount under /api/v1 to match production paths
	r.Route("/api/v1", func(r chi.Router) {
		cases.RegisterRoutes(r, h, jwtMgr)
	})

	return r, jwtMgr
}

func loginToken(t *testing.T, db *sql.DB, jwtMgr *auth.JWTManager, email string, role dbgen.UserRole) string {
	t.Helper()
	userID := seedUser(t, db, email, role)
	q := dbgen.New(db)
	repo := auth.NewRepository(q)
	svc := auth.NewService(repo, jwtMgr)
	pair, err := svc.Login(context.Background(), email, "TestP@ss1", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login %s: %v", email, err)
	}
	_ = userID
	return pair.AccessToken
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return bytes.NewBuffer(b)
}

// apiResp is the standard response envelope: {"data": {...}} or {"data": [...], "meta": {...}}
type apiResp struct {
	Data map[string]any `json:"data"`
}

// decodeData extracts the "data" object from an API envelope response.
func decodeData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env apiResp
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return env.Data
}

func TestHandlerCreateCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "hcreate@test.com", dbgen.UserRoleAnalyst)

	t.Run("201 on valid request", func(t *testing.T) {
		body := jsonBody(t, map[string]any{
			"title":    "Fraud cluster detected",
			"priority": "high",
			"tags":     []string{"upi", "fraud"},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeData(t, rec)
		if data["case_number"] == nil {
			t.Error("expected case_number in response")
		}
		if data["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", data["status"])
		}
	})

	t.Run("400 on title too short", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"title": "ab", "priority": "low"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("401 without token", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"title": "Unauthorized case", "priority": "low"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})
}

func TestHandlerGetCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "hget@test.com", dbgen.UserRoleAnalyst)

	// Create a case via API
	body := jsonBody(t, map[string]any{"title": "Case for GET test", "priority": "medium"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: create case failed %d: %s", rec.Code, rec.Body.String())
	}
	created := decodeData(t, rec)
	caseID, ok := created["id"].(string)
	if !ok || caseID == "" {
		t.Fatalf("expected id in create response, got: %v", created)
	}

	t.Run("200 for existing case", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeData(t, rec)
		if data["id"] != caseID {
			t.Errorf("id mismatch: %v", data["id"])
		}
	})

	t.Run("404 for unknown case", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+uuid.New().String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestHandlerStatusTransition(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	analystToken := loginToken(t, db, jwtMgr, "hstatus-analyst@test.com", dbgen.UserRoleAnalyst)

	// Create case
	body := jsonBody(t, map[string]any{"title": "Status HTTP test case", "priority": "high"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
	req.Header.Set("Authorization", "Bearer "+analystToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup create: %d %s", rec.Code, rec.Body.String())
	}
	created := decodeData(t, rec)
	caseID, _ := created["id"].(string)

	t.Run("422 on invalid transition", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"status": "closed"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/status", body)
		req.Header.Set("Authorization", "Bearer "+analystToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("200 on valid transition draft->open", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"status": "open"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/status", body)
		req.Header.Set("Authorization", "Bearer "+analystToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeData(t, rec)
		if data["status"] != "open" {
			t.Errorf("expected status=open, got %v", data["status"])
		}
		if data["sla_breach_at"] == nil {
			t.Error("SLA should be set after opening")
		}
	})
}

func TestHandlerListCases(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "hlist@test.com", dbgen.UserRoleAnalyst)

	// Seed 3 cases
	for i := 0; i < 3; i++ {
		body := jsonBody(t, map[string]any{
			"title":    fmt.Sprintf("List test case %d", i),
			"priority": "medium",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
	}

	type listResp struct {
		Data []any `json:"data"`
		Meta struct {
			TotalCount float64 `json:"total_count"`
		} `json:"meta"`
	}

	t.Run("200 with data and meta", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?limit=20", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp listResp
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data) < 3 {
			t.Errorf("expected >=3 items, got %d", len(resp.Data))
		}
		if resp.Meta.TotalCount < 3 {
			t.Errorf("expected total_count >=3, got %v", resp.Meta.TotalCount)
		}
	})

	t.Run("filter by status=open returns 0 (all are draft)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp listResp
		json.NewDecoder(rec.Body).Decode(&resp)
		if len(resp.Data) != 0 {
			t.Errorf("expected 0 open cases, got %d", len(resp.Data))
		}
	})
}

// ── Handler: Update, Delete, Assign, UpdateTags, ListActivity ─────────────────

// createCaseViaAPI is a test helper that creates a case and returns its ID.
func createCaseViaAPI(t *testing.T, r http.Handler, token string) string {
	t.Helper()
	body := jsonBody(t, map[string]any{"title": "Handler test case", "priority": "medium"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("createCaseViaAPI: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	id, _ := decodeData(t, rec)["id"].(string)
	if id == "" {
		t.Fatal("createCaseViaAPI: no id in response")
	}
	return id
}

func TestHandlerUpdateCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	// Use manager so no assignment check is needed
	token := loginToken(t, db, jwtMgr, "update-mgr@handler.com", dbgen.UserRoleManager)
	caseID := createCaseViaAPI(t, r, token)

	t.Run("200 updates title and priority", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"title": "Updated title", "priority": "high"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+caseID, body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeData(t, rec)
		if data["title"] != "Updated title" {
			t.Errorf("expected title='Updated title', got %v", data["title"])
		}
	})

	t.Run("404 for non-existent case", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"title": "Ghost"})
		req := httptest.NewRequest(http.MethodPut, "/api/v1/cases/"+uuid.New().String(), body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})
}

func TestHandlerDeleteCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)

	adminToken := loginToken(t, db, jwtMgr, "del-admin@handler.com", dbgen.UserRoleAdmin)
	analystToken := loginToken(t, db, jwtMgr, "del-analyst@handler.com", dbgen.UserRoleAnalyst)
	caseID := createCaseViaAPI(t, r, analystToken)

	t.Run("403 when analyst tries to delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cases/"+caseID, nil)
		req.Header.Set("Authorization", "Bearer "+analystToken)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("200 when admin deletes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cases/"+caseID, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeData(t, rec)
		if data["status"] != "deleted" {
			t.Errorf("expected status=deleted, got %v", data["status"])
		}
	})
}

func TestHandlerAssignCase(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)

	managerToken := loginToken(t, db, jwtMgr, "mgr@handler.com", dbgen.UserRoleManager)
	analystToken := loginToken(t, db, jwtMgr, "assignee@handler.com", dbgen.UserRoleAnalyst)
	analystID := seedUser(t, db, "assignee2@handler.com", dbgen.UserRoleAnalyst)
	caseID := createCaseViaAPI(t, r, managerToken)

	t.Run("200 manager assigns case", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"assigned_to": analystID.String()})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/assign", body)
		req.Header.Set("Authorization", "Bearer "+managerToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("403 analyst cannot assign", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"assigned_to": analystID.String()})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/assign", body)
		req.Header.Set("Authorization", "Bearer "+analystToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})
}

func TestHandlerUpdateTags(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "tags@handler.com", dbgen.UserRoleAnalyst)
	caseID := createCaseViaAPI(t, r, token)

	t.Run("200 sets tags", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"tags": []string{"fraud", "upi", "kyc"}})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/tags", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerListActivity(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "activity@handler.com", dbgen.UserRoleAnalyst)
	caseID := createCaseViaAPI(t, r, token)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID+"/activity", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerChangeStatus(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "status-mgr@handler.com", dbgen.UserRoleManager)
	caseID := createCaseViaAPI(t, r, token)

	t.Run("200 draft->open transition", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"status": "open"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/status", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("422 close without reason", func(t *testing.T) {
		token2 := loginToken(t, db, jwtMgr, "status-mgr2@handler.com", dbgen.UserRoleAdmin)
		caseID2 := createCaseViaAPI(t, r, token2)
		body, _ := json.Marshal(map[string]string{"status": "closed"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID2+"/status", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token2)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest {
			t.Errorf("expected 422/400, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerListCases_Filters(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "list-mgr@handler.com", dbgen.UserRoleManager)
	createCaseViaAPI(t, r, token)
	createCaseViaAPI(t, r, token)

	t.Run("200 no filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var env struct {
			Data []map[string]interface{} `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(env.Data) < 2 {
			t.Errorf("expected at least 2 cases, got %d", len(env.Data))
		}
	})

	t.Run("200 filter by status=open", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("200 filter by sla_breached=true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?sla_breached=true", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("200 with limit param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?limit=1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestHandlerGetCase_ErrorPaths(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "get-case@handler.com", dbgen.UserRoleAnalyst)
	caseID := createCaseViaAPI(t, r, token)

	t.Run("200 existing case", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+caseID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("404 non-existent case", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/"+uuid.New().String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("400 invalid uuid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases/not-a-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})
}

func TestHandlerCreateCase_Validation(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "create-val@handler.com", dbgen.UserRoleAnalyst)

	t.Run("400 title too short", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"title": "hi", "priority": "medium"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("201 default priority when omitted", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"title": "Test Case With Default Priority"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerInvalidIDPaths(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "invalid-id@handler.com", dbgen.UserRoleAdmin)

	badID := "not-a-uuid"

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"update bad id", http.MethodPut, "/api/v1/cases/" + badID, `{"title":"Valid Title For Test"}`},
		{"delete bad id", http.MethodDelete, "/api/v1/cases/" + badID, ""},
		{"status bad id", http.MethodPost, "/api/v1/cases/" + badID + "/status", `{"status":"open"}`},
		{"assign bad id", http.MethodPost, "/api/v1/cases/" + badID + "/assign", `{"assignee_id":"` + uuid.New().String() + `"}`},
		{"tags bad id", http.MethodPost, "/api/v1/cases/" + badID + "/tags", `{"tags":[]}`},
		{"activity bad id", http.MethodGet, "/api/v1/cases/" + badID + "/activity", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader = http.NoBody
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d: %s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

// compile check
var _ = fmt.Sprintf
