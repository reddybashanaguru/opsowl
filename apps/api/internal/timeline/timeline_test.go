package timeline_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
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
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

// ── infrastructure ──────────────────────────────────────────────────────────

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

func setupRouter(t *testing.T, db *sql.DB) (*chi.Mux, *auth.JWTManager) {
	t.Helper()

	jwtMgr, err := auth.NewJWTManager("../../keys/private.pem", "../../keys/public.pem", time.Hour)
	if err != nil {
		t.Fatalf("jwt manager: %v", err)
	}

	q := dbgen.New(db)
	hub := timeline.NewHub()
	repo := timeline.NewRepository(q)
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(repo, emitter)
	h := timeline.NewHandler(svc, hub, jwtMgr)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		timeline.RegisterRoutes(r, h, jwtMgr)
	})
	return r, jwtMgr
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
		 VALUES ('OWL-' || nextval('case_number_seq'), 'Test Case', '', 'draft', 'medium', $1, $2, '{}')
		 RETURNING id`,
		creatorID, orgID,
	).Scan(&caseID)
	if err != nil {
		t.Fatalf("seed case: %v", err)
	}
	return caseID
}

func seedEvent(t *testing.T, db *sql.DB, caseID, actorID uuid.UUID, evtType dbgen.TimelineEventType, occurredAt time.Time) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := db.QueryRowContext(context.Background(),
		`INSERT INTO timeline_events (case_id, event_type, source, actor_id, occurred_at, payload, metadata)
		 VALUES ($1, $2, 'user', $3, $4, '{}', '{}') RETURNING id`,
		caseID, evtType, actorID, occurredAt,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed event: %v", err)
	}
	return id
}

func makeToken(t *testing.T, jwtMgr *auth.JWTManager, userID uuid.UUID, email string, role dbgen.UserRole) string {
	t.Helper()
	tok, err := jwtMgr.Sign(userID, email, role)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tok
}

// decodeCursor mirrors the base64(JSON{occurred_at,id}) format used internally.
func decodeCursor(cursor string) (*time.Time, *uuid.UUID, error) {
	raw, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, nil, err
	}
	var p struct {
		OccurredAt time.Time `json:"occurred_at"`
		ID         uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil, err
	}
	return &p.OccurredAt, &p.ID, nil
}

// ── service-layer tests ──────────────────────────────────────────────────────

func TestServiceList_Empty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "v1@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	events, meta, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
	if meta.HasMore {
		t.Error("expected HasMore=false on empty list")
	}
}

func TestServiceList_OrderedNewestFirst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "v2@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)

	now := time.Now().UTC().Truncate(time.Second)
	e1 := seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseCreated, now.Add(-2*time.Hour))
	e2 := seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseStatusChanged, now.Add(-1*time.Hour))
	e3 := seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseAssigned, now)
	_ = e2

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	events, _, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].ID != e3 {
		t.Errorf("expected newest first: got %s, want %s", events[0].ID, e3)
	}
	if events[2].ID != e1 {
		t.Errorf("expected oldest last: got %s, want %s", events[2].ID, e1)
	}
}

func TestServiceList_FilterBySource(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "v3@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)

	now := time.Now().UTC()
	seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseCreated, now)
	// Insert a system event directly
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO timeline_events (case_id, event_type, source, occurred_at, payload, metadata)
		 VALUES ($1, 'sla_warning', 'system', $2, '{}', '{}')`,
		caseID, now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("insert system event: %v", err)
	}

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	src := dbgen.TimelineEventSourceUser
	events, _, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 50, Source: &src})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 user event, got %d", len(events))
	}
	if events[0].Source != dbgen.TimelineEventSourceUser {
		t.Errorf("unexpected source: %s", events[0].Source)
	}
}

func TestServiceList_FilterByEventType(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "v4@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)

	now := time.Now().UTC()
	seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseCreated, now)
	seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseAssigned, now.Add(time.Minute))

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	et := dbgen.TimelineEventTypeCaseCreated
	events, _, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 50, EventType: &et})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 1 || events[0].EventType != dbgen.TimelineEventTypeCaseCreated {
		t.Errorf("filter by event_type failed: got %d events", len(events))
	}
}

func TestServiceList_Pagination_Cursor(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "v5@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseCreated, now.Add(time.Duration(i)*time.Minute))
	}

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	// Page 1 — 3 events
	page1, meta1, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 3})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 on page1, got %d", len(page1))
	}
	if !meta1.HasMore || meta1.NextCursor == nil {
		t.Fatal("expected HasMore=true and NextCursor on page1")
	}

	// Decode cursor and fetch page 2
	ct, cid, err := decodeCursor(*meta1.NextCursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	page2, meta2, err := svc.List(context.Background(), caseID, timeline.ListFilters{
		Limit: 3, CursorTime: ct, CursorID: cid,
	})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("expected 2 on page2, got %d", len(page2))
	}
	if meta2.HasMore {
		t.Error("expected HasMore=false on last page")
	}
	// Ensure no duplicate IDs across pages
	seen := map[uuid.UUID]bool{}
	for _, e := range page1 {
		seen[e.ID] = true
	}
	for _, e := range page2 {
		if seen[e.ID] {
			t.Errorf("duplicate event ID across pages: %s", e.ID)
		}
	}
}

func TestServiceList_BackDatedFlag(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "v6@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)

	// occurred_at is 3 hours ago; created_at will be ~now
	oldTime := time.Now().UTC().Add(-3 * time.Hour)
	seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseCreated, oldTime)

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	events, _, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected 1 event")
	}
	if !events[0].BackDated {
		t.Error("expected back_dated=true when |occurred_at - created_at| > 1h")
	}
}

func TestServiceList_FilterByActor(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	actor1 := seedUser(t, db, "actor1@test.com", dbgen.UserRoleAnalyst)
	actor2 := seedUser(t, db, "actor2@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, actor1)

	now := time.Now().UTC()
	seedEvent(t, db, caseID, actor1, dbgen.TimelineEventTypeCaseCreated, now)
	seedEvent(t, db, caseID, actor2, dbgen.TimelineEventTypeCaseAssigned, now.Add(time.Minute))

	q := dbgen.New(db)
	hub := timeline.NewHub()
	emitter := timeline.NewEmitter(q, hub)
	svc := timeline.NewService(timeline.NewRepository(q), emitter)

	events, _, err := svc.List(context.Background(), caseID, timeline.ListFilters{Limit: 50, ActorID: &actor1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event for actor1, got %d", len(events))
	}
}

// ── handler-layer tests ──────────────────────────────────────────────────────

func TestHandlerListTimeline_200(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "h1@test.com", dbgen.UserRoleViewer)
	caseID := seedCase(t, db, userID)
	seedEvent(t, db, caseID, userID, dbgen.TimelineEventTypeCaseCreated, time.Now())

	r, jwtMgr := setupRouter(t, db)
	tok := makeToken(t, jwtMgr, userID, "h1@test.com", dbgen.UserRoleViewer)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/cases/%s/timeline", caseID), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var env struct {
		Data []map[string]any `json:"data"`
		Meta map[string]any   `json:"meta"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(env.Data) != 1 {
		t.Errorf("expected 1 event, got %d", len(env.Data))
	}
	if env.Meta == nil {
		t.Error("expected meta in response")
	}
}

func TestHandlerListTimeline_401_NoToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/cases/%s/timeline", uuid.New()), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerCreateEvent_201_Analyst(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "h2@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)
	r, jwtMgr := setupRouter(t, db)
	tok := makeToken(t, jwtMgr, userID, "h2@test.com", dbgen.UserRoleAnalyst)

	body, _ := json.Marshal(map[string]any{
		"event_type":  "note_added",
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
		"payload":     map[string]any{"note": "test note"},
	})
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/cases/%s/timeline/events", caseID),
		bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateEvent_400_DisallowedType(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "h3@test.com", dbgen.UserRoleAnalyst)
	caseID := seedCase(t, db, userID)
	r, jwtMgr := setupRouter(t, db)
	tok := makeToken(t, jwtMgr, userID, "h3@test.com", dbgen.UserRoleAnalyst)

	body, _ := json.Marshal(map[string]any{
		"event_type":  "case_created", // system-only type
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/cases/%s/timeline/events", caseID),
		bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateEvent_403_ViewerForbidden(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()

	userID := seedUser(t, db, "h4@test.com", dbgen.UserRoleViewer)
	caseID := seedCase(t, db, userID)
	r, jwtMgr := setupRouter(t, db)
	tok := makeToken(t, jwtMgr, userID, "h4@test.com", dbgen.UserRoleViewer)

	body, _ := json.Marshal(map[string]any{
		"event_type":  "note_added",
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/v1/cases/%s/timeline/events", caseID),
		bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerStream_401_MissingToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/cases/%s/timeline/stream", uuid.New()), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerStream_401_InvalidToken(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	r, _ := setupRouter(t, db)

	req := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/cases/%s/timeline/stream?token=bad.token", uuid.New()), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
