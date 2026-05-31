package subjects

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

// ── In-process unit tests (no DB) ────────────────────────────────────────────
// These test the service layer logic without requiring a live database.

// mockRepo implements just enough of Repository for service-level tests.
type mockRepo struct {
	subjects map[uuid.UUID]*SubjectResponse
	byID     map[string]*SubjectResponse // "idType:idNumber" → subject
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		subjects: map[uuid.UUID]*SubjectResponse{},
		byID:     map[string]*SubjectResponse{},
	}
}

func (m *mockRepo) Create(_ context.Context, _, _ uuid.UUID, req CreateSubjectRequest) (*SubjectResponse, error) {
	id := uuid.New()
	s := &SubjectResponse{
		ID:            id,
		CanonicalName: req.CanonicalName,
		EntityType:    req.EntityType,
		IDType:        req.IDType,
		IDNumber:      req.IDNumber,
	}
	m.subjects[id] = s
	if req.IDType != "" && req.IDNumber != "" {
		m.byID[req.IDType+":"+req.IDNumber] = s
	}
	return s, nil
}

func (m *mockRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*SubjectResponse, error) {
	s, ok := m.subjects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *mockRepo) GetByIdentity(_ context.Context, _ uuid.UUID, idType, idNumber string) (*SubjectResponse, error) {
	s, ok := m.byID[idType+":"+idNumber]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *mockRepo) List(_ context.Context, _ uuid.UUID, _ ListFilters) ([]SubjectResponse, int64, error) {
	out := make([]SubjectResponse, 0, len(m.subjects))
	for _, s := range m.subjects {
		out = append(out, *s)
	}
	return out, int64(len(out)), nil
}

func (m *mockRepo) Update(_ context.Context, id, _ uuid.UUID, req UpdateSubjectRequest) (*SubjectResponse, error) {
	s, ok := m.subjects[id]
	if !ok {
		return nil, ErrNotFound
	}
	if req.CanonicalName != nil {
		s.CanonicalName = *req.CanonicalName
	}
	if req.IsWatchlisted != nil {
		s.IsWatchlisted = *req.IsWatchlisted
	}
	if req.RiskScore != nil {
		s.RiskScore = *req.RiskScore
	}
	return s, nil
}

func (m *mockRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	if _, ok := m.subjects[id]; !ok {
		return ErrNotFound
	}
	delete(m.subjects, id)
	return nil
}

func (m *mockRepo) LinkToCase(_ context.Context, _, _ uuid.UUID, _ uuid.UUID, _ string) error {
	return nil
}

func (m *mockRepo) UnlinkFromCase(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockRepo) ListForCase(_ context.Context, _ uuid.UUID) ([]CaseSubjectRef, error) {
	return []CaseSubjectRef{}, nil
}

func (m *mockRepo) ListCasesForSubject(_ context.Context, _ uuid.UUID) ([]SubjectCaseRef, error) {
	return []SubjectCaseRef{}, nil
}

func (m *mockRepo) AppendRiskHistory(_ context.Context, _ uuid.UUID, _ int, _, _, _ string) error {
	return nil
}

func (m *mockRepo) RiskHistory(_ context.Context, _ uuid.UUID) ([]RiskHistoryEntry, error) {
	return []RiskHistoryEntry{}, nil
}

func (m *mockRepo) ListFilingsForSubject(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]map[string]any, error) {
	return []map[string]any{}, nil
}

// Service needs a *Repository not an interface. Wire a real repository backed by
// a mock so we test the service logic through its actual methods.
// We use a thin adapter approach — the service calls r.repo.* — so we need a
// sql.DB-backed repo but can test it without a live DB by using a separate
// mock-based service.

// serviceWithMock builds a service that delegates to the mock instead of
// the real repository. We use this to test service-layer logic in isolation.
type serviceWithMock struct {
	repo *mockRepo
}

func (s *serviceWithMock) Create(ctx context.Context, claims *auth.Claims, req CreateSubjectRequest) (*SubjectResponse, error) {
	if req.CanonicalName == "" {
		return nil, ErrNameRequired
	}
	return s.repo.Create(ctx, defaultOrgID, claims.UserID, req)
}

func (s *serviceWithMock) ResolveOrCreate(ctx context.Context, claims *auth.Claims, req CreateSubjectRequest) (*SubjectResponse, bool, error) {
	if req.IDType != "" && req.IDNumber != "" {
		existing, err := s.repo.GetByIdentity(ctx, defaultOrgID, req.IDType, req.IDNumber)
		if err == nil {
			return existing, false, nil
		}
	}
	sub, err := s.repo.Create(ctx, defaultOrgID, claims.UserID, req)
	if err != nil {
		return nil, false, err
	}
	return sub, true, nil
}

func (s *serviceWithMock) SetWatchlist(ctx context.Context, _ *auth.Claims, id uuid.UUID, on bool) (*SubjectResponse, error) {
	return s.repo.Update(ctx, id, defaultOrgID, UpdateSubjectRequest{IsWatchlisted: &on})
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func makeClaims() *auth.Claims {
	return &auth.Claims{
		UserID: uuid.New(),
		Email:  "test@opsowl.dev",
		Role:   db.UserRoleAnalyst,
	}
}

func TestCreateSubjectNameRequired(t *testing.T) {
	svc := &serviceWithMock{repo: newMockRepo()}
	_, err := svc.Create(context.Background(), makeClaims(), CreateSubjectRequest{})
	if err != ErrNameRequired {
		t.Errorf("got %v, want ErrNameRequired", err)
	}
}

func TestCreateSubject(t *testing.T) {
	svc := &serviceWithMock{repo: newMockRepo()}
	sub, err := svc.Create(context.Background(), makeClaims(), CreateSubjectRequest{
		CanonicalName: "John Doe",
		IDType:        "PASSPORT",
		IDNumber:      "AB123456",
		EntityType:    "individual",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sub.CanonicalName != "John Doe" {
		t.Errorf("name got %q, want John Doe", sub.CanonicalName)
	}
	if sub.ID == uuid.Nil {
		t.Error("ID should not be nil")
	}
}

func TestResolveOrCreate_NewSubject(t *testing.T) {
	svc := &serviceWithMock{repo: newMockRepo()}
	claims := makeClaims()

	sub, created, err := svc.ResolveOrCreate(context.Background(), claims, CreateSubjectRequest{
		CanonicalName: "Jane Smith",
		IDType:        "NATIONAL_ID",
		IDNumber:      "KE-001234",
		EntityType:    "individual",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !created {
		t.Error("expected subject to be created (new)")
	}
	if sub.CanonicalName != "Jane Smith" {
		t.Errorf("name got %q", sub.CanonicalName)
	}
}

func TestResolveOrCreate_ExistingSubject(t *testing.T) {
	svc := &serviceWithMock{repo: newMockRepo()}
	claims := makeClaims()
	req := CreateSubjectRequest{
		CanonicalName: "Existing Subject",
		IDType:        "TAX_ID",
		IDNumber:      "US-987654321",
		EntityType:    "individual",
	}

	// Create once
	first, _, _ := svc.ResolveOrCreate(context.Background(), claims, req)

	// Resolve again — should return existing, not create new
	second, created, err := svc.ResolveOrCreate(context.Background(), claims, req)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if created {
		t.Error("expected subject to be resolved (existing), not created")
	}
	if first.ID != second.ID {
		t.Errorf("ID mismatch: first %v, second %v", first.ID, second.ID)
	}
}

func TestSetWatchlist(t *testing.T) {
	svc := &serviceWithMock{repo: newMockRepo()}
	claims := makeClaims()

	sub, _ := svc.Create(context.Background(), claims, CreateSubjectRequest{
		CanonicalName: "Watch Me",
	})

	updated, err := svc.SetWatchlist(context.Background(), claims, sub.ID, true)
	if err != nil {
		t.Fatalf("watchlist: %v", err)
	}
	if !updated.IsWatchlisted {
		t.Error("expected IsWatchlisted to be true")
	}

	updated2, _ := svc.SetWatchlist(context.Background(), claims, sub.ID, false)
	if updated2.IsWatchlisted {
		t.Error("expected IsWatchlisted to be false after removal")
	}
}

func TestResolveOrCreate_NoIDFallsBackToCreate(t *testing.T) {
	svc := &serviceWithMock{repo: newMockRepo()}
	claims := makeClaims()

	// Without id_type or id_number, always creates a new record
	s1, c1, _ := svc.ResolveOrCreate(context.Background(), claims, CreateSubjectRequest{
		CanonicalName: "No ID Subject",
	})
	s2, c2, _ := svc.ResolveOrCreate(context.Background(), claims, CreateSubjectRequest{
		CanonicalName: "No ID Subject",
	})
	if !c1 || !c2 {
		t.Error("expected both to be created (no ID to dedup on)")
	}
	if s1.ID == s2.ID {
		t.Error("expected different IDs when no dedup key available")
	}
}

// ── DB integration tests (skipped unless TEST_DB_URL is set) ─────────────────

func TestSubjectsDBIntegration(t *testing.T) {
	dsn := getTestDSN()
	if dsn == "" {
		t.Skip("TEST_DB_URL not set — skipping DB integration tests")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Skipf("DB unreachable: %v", err)
	}

	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()
	claims := makeClaims()

	// Create
	sub, err := svc.Create(ctx, claims, CreateSubjectRequest{
		CanonicalName: "Integration Test Subject",
		IDType:        "PASSPORT",
		IDNumber:      "IT-TEST-" + uuid.New().String()[:8],
		EntityType:    "individual",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Logf("created subject %s", sub.ID)

	// Get
	fetched, err := svc.Get(ctx, sub.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.CanonicalName != sub.CanonicalName {
		t.Errorf("name mismatch: got %q, want %q", fetched.CanonicalName, sub.CanonicalName)
	}

	// Update
	newName := "Integration Test Subject Updated"
	updated, err := svc.Update(ctx, claims, sub.ID, UpdateSubjectRequest{CanonicalName: &newName})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.CanonicalName != newName {
		t.Errorf("updated name got %q, want %q", updated.CanonicalName, newName)
	}

	// ResolveOrCreate dedup
	existing, created, err := svc.ResolveOrCreate(ctx, claims, CreateSubjectRequest{
		CanonicalName: "Anything",
		IDType:        sub.IDType,
		IDNumber:      sub.IDNumber,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if created {
		t.Error("expected existing subject, not new create")
	}
	if existing.ID != sub.ID {
		t.Errorf("ID mismatch: got %v, want %v", existing.ID, sub.ID)
	}

	// Watchlist
	watched, err := svc.SetWatchlist(ctx, claims, sub.ID, true)
	if err != nil {
		t.Fatalf("watchlist: %v", err)
	}
	if !watched.IsWatchlisted {
		t.Error("expected watchlisted")
	}

	// Delete
	if err := svc.Delete(ctx, claims, sub.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func getTestDSN() string {
	return os.Getenv("TEST_DB_URL")
}
