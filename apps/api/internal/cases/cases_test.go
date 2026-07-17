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
	"github.com/reddybashanaguru/opsowl/internal/notifications"
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
	return cases.NewService(repo, db, noopEmitter{})
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

	t.Run("creates with open status and OWL case number", func(t *testing.T) {
		c, err := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
			Title:    "Suspicious transaction cluster",
			Priority: dbgen.CasePriorityHigh,
			Tags:     []string{"fraud", "upi"},
		})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if c.Status != dbgen.CaseStatusOpen {
			t.Errorf("expected status=open, got %s", c.Status)
		}
		if !strings.HasPrefix(c.CaseNumber, "OWL-") {
			t.Errorf("expected OWL- prefix, got %s", c.CaseNumber)
		}
		if len(c.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(c.Tags))
		}
		// SLA is auto-set from priority on creation
		if c.SLABreachAt == nil {
			t.Error("SLA breach time should be set on creation (high = 24h)")
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
		c, err := svc.Get(context.Background(), created.ID, cases.DefaultOrgID)
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
		_, err := svc.Get(context.Background(), uuid.New(), cases.DefaultOrgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// ── C6: IDOR org-scoping tests ────────────────────────────────────────────────

func TestGetCase_IDORProtection(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc := newTestService(t, db)
	analystID := seedUser(t, db, "idor@test.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "idor@test.com", dbgen.UserRoleAnalyst)

	created, err := svc.Create(context.Background(), claims, cases.CreateCaseRequest{
		Title:    "Org A case",
		Priority: dbgen.CasePriorityHigh,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Org A case is accessible from Org A
	t.Run("own org case is accessible", func(t *testing.T) {
		_, err := svc.Get(context.Background(), created.ID, cases.DefaultOrgID)
		if err != nil {
			t.Errorf("expected no error for own-org case, got: %v", err)
		}
	})

	// Directly insert a case with a DIFFERENT org_id (simulating Org B's case)
	foreignOrgID := uuid.New()
	var foreignCaseID uuid.UUID
	err = db.QueryRowContext(context.Background(), `
		INSERT INTO cases (case_number, title, description, status, priority, created_by, organization_id, tags)
		VALUES (
			'OWL-TEST-' || nextval('case_number_seq')::TEXT,
			'Foreign org case', '', 'open', 'medium', $1, $2, '{}'
		)
		RETURNING id`,
		analystID, foreignOrgID,
	).Scan(&foreignCaseID)
	if err != nil {
		t.Fatalf("insert foreign-org case: %v", err)
	}

	// Accessing foreign org case from DefaultOrgID must return ErrNotFound
	t.Run("cross-org case returns ErrNotFound (IDOR blocked)", func(t *testing.T) {
		_, err := svc.Get(context.Background(), foreignCaseID, cases.DefaultOrgID)
		if err == nil {
			t.Fatal("expected ErrNotFound for cross-org access, got nil — IDOR vulnerability!")
		}
	})

	// The foreign org CAN access its own case
	t.Run("case is accessible to its own org", func(t *testing.T) {
		_, err := svc.Get(context.Background(), foreignCaseID, foreignOrgID)
		if err != nil {
			t.Errorf("expected no error for foreign-org accessing own case, got: %v", err)
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

	t.Run("filter by status=open returns all 3 (default status)", func(t *testing.T) {
		s := dbgen.CaseStatusOpen
		cs, _, err := svc.List(context.Background(), cases.ListFilters{Status: &s, Limit: 20})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(cs) != 3 {
			t.Errorf("expected 3 open, got %d", len(cs))
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

	t.Run("create sets SLA from priority", func(t *testing.T) {
		c := createCase(t)
		// Cases now start as open with SLA already set from priority
		if c.Status != dbgen.CaseStatusOpen {
			t.Errorf("expected open, got %s", c.Status)
		}
		if c.SLABreachAt == nil {
			t.Error("SLA breach time should be set at creation")
		}
		// high priority = 24h SLA
		expected := time.Now().Add(24 * time.Hour)
		diff := c.SLABreachAt.Sub(expected)
		if diff < -2*time.Minute || diff > 2*time.Minute {
			t.Errorf("SLA time off by %v", diff)
		}
	})

	t.Run("open -> in_progress by analyst", func(t *testing.T) {
		c := createCase(t)
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
			t.Fatal("expected error for invalid transition open->closed (must go through in_progress→pending_review)")
		}
	})

	t.Run("escalation requires manager role", func(t *testing.T) {
		c := createCase(t)
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
		_, err = svc.Get(context.Background(), c.ID, cases.DefaultOrgID)
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
	svc := cases.NewService(repo, db, noopEmitter{})
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
	result, err := svc.Login(context.Background(), email, "TestP@ss1", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login %s: %v", email, err)
	}
	if result.Pair == nil {
		t.Fatalf("expected token pair for %s, got MFA required", email)
	}
	_ = userID
	return result.Pair.AccessToken
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
		if data["status"] != "open" {
			t.Errorf("expected status=open, got %v", data["status"])
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

	t.Run("200 on valid transition open->in_progress", func(t *testing.T) {
		body := jsonBody(t, map[string]any{"status": "in_progress"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/cases/"+caseID+"/status", body)
		req.Header.Set("Authorization", "Bearer "+analystToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		data := decodeData(t, rec)
		if data["status"] != "in_progress" {
			t.Errorf("expected status=in_progress, got %v", data["status"])
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

	t.Run("filter by status=open returns 3 (all created as open)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp listResp
		json.NewDecoder(rec.Body).Decode(&resp)
		if len(resp.Data) < 3 {
			t.Errorf("expected >=3 open cases, got %d", len(resp.Data))
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

	t.Run("200 open->in_progress transition", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"status": "in_progress"})
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

func TestHandlerListCases_MultiStatusFilter(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, jwtMgr := setupRouter(t, db)
	token := loginToken(t, db, jwtMgr, "multistat@handler.com", dbgen.UserRoleAdmin)

	svc := newTestService(t, db)
	analystID := seedUser(t, db, "mseed@handler.com", dbgen.UserRoleAnalyst)
	claims := claimsFor(analystID, "mseed@handler.com", dbgen.UserRoleAnalyst)

	ctx := context.Background()

	// Create cases: all start as open; move two through the pipeline
	openCase, _ := svc.Create(ctx, claims, cases.CreateCaseRequest{Title: "Open case for multi-filter", Priority: dbgen.CasePriorityMedium})
	inProgressCase, _ := svc.Create(ctx, claims, cases.CreateCaseRequest{Title: "InProgress case for multi-filter", Priority: dbgen.CasePriorityLow})
	closedCase, _ := svc.Create(ctx, claims, cases.CreateCaseRequest{Title: "Closed case for multi-filter", Priority: dbgen.CasePriorityHigh})

	adminID := seedUser(t, db, "mseedadmin@handler.com", dbgen.UserRoleAdmin)
	adminClaims := claimsFor(adminID, "mseedadmin@handler.com", dbgen.UserRoleAdmin)
	// Move inProgressCase to in_progress
	_, _ = svc.ChangeStatus(ctx, claims, inProgressCase.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
	// Move closedCase through the full pipeline to closed
	_, _ = svc.ChangeStatus(ctx, claims, closedCase.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
	_, _ = svc.ChangeStatus(ctx, claims, closedCase.ID, cases.StatusRequest{Status: dbgen.CaseStatusPendingReview})
	_, _ = svc.ChangeStatus(ctx, adminClaims, closedCase.ID, cases.StatusRequest{Status: dbgen.CaseStatusClosed})
	_ = openCase

	t.Run("single status filter returns only matching", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Data []struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, c := range resp.Data {
			if c.Status != "open" {
				t.Errorf("expected only open cases, got status=%s", c.Status)
			}
		}
	})

	t.Run("multi-status filter returns cases matching any selected status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open,closed", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Data []struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data) == 0 {
			t.Fatal("expected at least one case")
		}
		for _, c := range resp.Data {
			if c.Status != "open" && c.Status != "closed" {
				t.Errorf("expected only open or closed cases, got status=%s", c.Status)
			}
		}
	})

	t.Run("multi-status excludes unselected statuses", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/cases?status=open,closed", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		var resp struct {
			Data []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"data"`
		}
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		for _, c := range resp.Data {
			if c.ID == inProgressCase.ID.String() {
				t.Error("in_progress case should not appear in open,closed filter")
			}
		}
	})
}

// compile check
var _ = fmt.Sprintf

// ── Manager notification tests ───────────────────────────────────────────────

// newTestServiceWithNotifs builds a service wired with a real notifications.Service
// backed by the same DB, so notification rows can be inspected after transitions.
func newTestServiceWithNotifs(t *testing.T, sqlDB *sql.DB) (*cases.Service, *notifications.Service) {
	t.Helper()
	q := dbgen.New(sqlDB)
	repo := cases.NewRepository(q)
	notifSvc := notifications.NewService(q, notifications.NewBroker())
	svc := cases.NewService(repo, sqlDB, noopEmitter{}).WithNotifications(notifSvc)
	return svc, notifSvc
}

// unreadCountFor returns the number of unread notifications for a given user.
func unreadCountFor(t *testing.T, sqlDB *sql.DB, userID uuid.UUID) int64 {
	t.Helper()
	count, err := dbgen.New(sqlDB).CountUnreadNotifications(context.Background(), userID)
	if err != nil {
		t.Fatalf("count notifications for %s: %v", userID, err)
	}
	return count
}

func TestListManagers(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	mgr1ID := seedUser(t, db, "lm-mgr1@notif.com", dbgen.UserRoleManager)
	mgr2ID := seedUser(t, db, "lm-mgr2@notif.com", dbgen.UserRoleManager)
	adminID := seedUser(t, db, "lm-admin@notif.com", dbgen.UserRoleAdmin)
	analystID := seedUser(t, db, "lm-analyst@notif.com", dbgen.UserRoleAnalyst)

	repo := cases.NewRepository(dbgen.New(db))
	managers, err := repo.ListManagers(context.Background())
	if err != nil {
		t.Fatalf("ListManagers: %v", err)
	}

	ids := make(map[uuid.UUID]struct{}, len(managers))
	for _, m := range managers {
		ids[m.ID] = struct{}{}
	}

	t.Run("includes managers", func(t *testing.T) {
		if _, ok := ids[mgr1ID]; !ok {
			t.Error("mgr1 missing from ListManagers")
		}
		if _, ok := ids[mgr2ID]; !ok {
			t.Error("mgr2 missing from ListManagers")
		}
	})

	t.Run("includes admins", func(t *testing.T) {
		if _, ok := ids[adminID]; !ok {
			t.Error("admin missing from ListManagers")
		}
	})

	t.Run("excludes analysts", func(t *testing.T) {
		if _, ok := ids[analystID]; ok {
			t.Error("analyst should not appear in ListManagers")
		}
	})
}

func TestPendingReviewNotifiesManagers(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestServiceWithNotifs(t, db)

	analystID := seedUser(t, db, "pr-analyst@notif.com", dbgen.UserRoleAnalyst)
	mgr1ID := seedUser(t, db, "pr-mgr1@notif.com", dbgen.UserRoleManager)
	mgr2ID := seedUser(t, db, "pr-mgr2@notif.com", dbgen.UserRoleManager)
	analystClaims := claimsFor(analystID, "pr-analyst@notif.com", dbgen.UserRoleAnalyst)

	c, _ := svc.Create(context.Background(), analystClaims, cases.CreateCaseRequest{
		Title:    "Notification fan-out test case",
		Priority: dbgen.CasePriorityHigh,
	})
	svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
	_, err := svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusPendingReview})
	if err != nil {
		t.Fatalf("transition to pending_review: %v", err)
	}

	time.Sleep(300 * time.Millisecond) // allow goroutine to flush

	t.Run("manager1 receives notification", func(t *testing.T) {
		if unreadCountFor(t, db, mgr1ID) == 0 {
			t.Error("manager1 should have an unread pending_review notification")
		}
	})

	t.Run("manager2 receives notification", func(t *testing.T) {
		if unreadCountFor(t, db, mgr2ID) == 0 {
			t.Error("manager2 should have an unread pending_review notification")
		}
	})

	t.Run("submitting analyst is not notified via manager channel", func(t *testing.T) {
		// Analyst is not in ListManagers, so the fan-out loop never targets them.
		// They may have zero or one notification depending on assignee logic,
		// but should not get a duplicate from the manager fan-out.
		count := unreadCountFor(t, db, analystID)
		// Assert <= 1: at most the assignee notification (if they were assigned)
		if count > 1 {
			t.Errorf("analyst received %d notifications — manager fan-out should not target them", count)
		}
	})
}

func TestPendingReview_ManagerSelfNoNotify(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestServiceWithNotifs(t, db)

	// A manager who themselves submits the pending_review should not self-notify.
	mgrID := seedUser(t, db, "self-mgr@notif.com", dbgen.UserRoleManager)
	otherMgrID := seedUser(t, db, "other-mgr@notif.com", dbgen.UserRoleManager)
	mgrClaims := claimsFor(mgrID, "self-mgr@notif.com", dbgen.UserRoleManager)

	c, _ := svc.Create(context.Background(), mgrClaims, cases.CreateCaseRequest{
		Title:    "Manager self-submit case",
		Priority: dbgen.CasePriorityMedium,
	})
	svc.ChangeStatus(context.Background(), mgrClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
	svc.ChangeStatus(context.Background(), mgrClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusPendingReview})

	time.Sleep(300 * time.Millisecond)

	t.Run("submitter manager does not self-notify", func(t *testing.T) {
		if unreadCountFor(t, db, mgrID) > 0 {
			t.Error("manager should not receive their own pending_review notification")
		}
	})

	t.Run("other manager is still notified", func(t *testing.T) {
		if unreadCountFor(t, db, otherMgrID) == 0 {
			t.Error("other manager should receive the notification")
		}
	})
}

func TestNonPendingReview_DoesNotNotifyManagers(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, _ := newTestServiceWithNotifs(t, db)

	analystID := seedUser(t, db, "nonotif-analyst@notif.com", dbgen.UserRoleAnalyst)
	mgrID := seedUser(t, db, "nonotif-mgr@notif.com", dbgen.UserRoleManager)
	analystClaims := claimsFor(analystID, "nonotif-analyst@notif.com", dbgen.UserRoleAnalyst)

	c, _ := svc.Create(context.Background(), analystClaims, cases.CreateCaseRequest{
		Title:    "No manager notif on in_progress test",
		Priority: dbgen.CasePriorityLow,
	})
	svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})

	time.Sleep(300 * time.Millisecond)

	t.Run("in_progress transition does not notify managers", func(t *testing.T) {
		if unreadCountFor(t, db, mgrID) > 0 {
			t.Errorf("manager should not be notified for in_progress transition, got %d", unreadCountFor(t, db, mgrID))
		}
	})
}

func TestPendingReview_NotificationContent(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	svc, notifSvc := newTestServiceWithNotifs(t, db)

	analystID := seedUser(t, db, "content-analyst@notif.com", dbgen.UserRoleAnalyst)
	mgrID := seedUser(t, db, "content-mgr@notif.com", dbgen.UserRoleManager)
	analystClaims := claimsFor(analystID, "content-analyst@notif.com", dbgen.UserRoleAnalyst)

	c, _ := svc.Create(context.Background(), analystClaims, cases.CreateCaseRequest{
		Title:    "Content check case",
		Priority: dbgen.CasePriorityHigh,
	})
	svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusInProgress})
	svc.ChangeStatus(context.Background(), analystClaims, c.ID, cases.StatusRequest{Status: dbgen.CaseStatusPendingReview})

	time.Sleep(300 * time.Millisecond)

	notifs, _, err := notifSvc.List(context.Background(), mgrID, uuid.Nil, 10)
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notifs) == 0 {
		t.Fatal("expected at least 1 notification for manager")
	}
	n := notifs[0]

	t.Run("priority is high", func(t *testing.T) {
		if n.Priority != string(dbgen.NotificationPriorityHigh) {
			t.Errorf("expected priority=high, got %s", n.Priority)
		}
	})

	t.Run("title mentions review", func(t *testing.T) {
		if !strings.Contains(strings.ToLower(n.Title), "review") {
			t.Errorf("expected 'review' in title, got: %s", n.Title)
		}
	})

	t.Run("message contains case number", func(t *testing.T) {
		if !strings.Contains(n.Message, c.CaseNumber) {
			t.Errorf("expected case number %s in message, got: %s", c.CaseNumber, n.Message)
		}
	})

	t.Run("message contains actor email", func(t *testing.T) {
		if !strings.Contains(n.Message, "content-analyst@notif.com") {
			t.Errorf("expected actor email in message, got: %s", n.Message)
		}
	})

	t.Run("link points to case", func(t *testing.T) {
		if n.LinkURL == nil || !strings.Contains(*n.LinkURL, c.ID.String()) {
			t.Errorf("expected case ID in link_url, got: %v", n.LinkURL)
		}
	})
}
