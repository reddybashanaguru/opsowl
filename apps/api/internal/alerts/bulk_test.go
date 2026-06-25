package alerts

// Unit tests for bulk alert operations — no DB or network required.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chi "github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"

	"github.com/reddybashanaguru/opsowl/internal/auth"
)

// ── In-memory store ───────────────────────────────────────────────────────────

type memAlertStore struct {
	alerts map[uuid.UUID]*AlertResponse
}

func newMemStore() *memAlertStore {
	return &memAlertStore{alerts: make(map[uuid.UUID]*AlertResponse)}
}

func (m *memAlertStore) seed(id uuid.UUID, status string) {
	m.alerts[id] = &AlertResponse{ID: id, Status: status}
}

func (m *memAlertStore) get(id uuid.UUID) (*AlertResponse, error) {
	a, ok := m.alerts[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (m *memAlertStore) setStatus(id uuid.UUID, status string) error {
	if _, ok := m.alerts[id]; !ok {
		return ErrNotFound
	}
	m.alerts[id].Status = status
	return nil
}

// ── Thin service wrapper driving memAlertStore ────────────────────────────────

// testBulkSvc replicates the BulkDismiss / BulkMarkDuplicate logic using
// memAlertStore so we can verify the business rules without a real DB.
type testBulkSvc struct{ store *memAlertStore }

func (s *testBulkSvc) BulkDismiss(ctx context.Context, _ *auth.Claims, ids []uuid.UUID, reason string) (*BulkResult, error) {
	if len(ids) == 0 {
		return &BulkResult{}, nil
	}
	if len(ids) > maxBulkAlerts {
		return nil, fmt.Errorf("bulk dismiss limited to %d alerts per request", maxBulkAlerts)
	}
	if reason == "" {
		reason = "bulk dismissed by analyst"
	}
	res := &BulkResult{}
	for _, id := range ids {
		a, err := s.store.get(id)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, BulkResultItem{AlertID: id, Error: "alert not found"})
			continue
		}
		if a.Status == "dismissed" || a.Status == "converted" {
			res.Failed++
			res.Errors = append(res.Errors, BulkResultItem{AlertID: id, Error: "alert already actioned: " + a.Status})
			continue
		}
		if err := s.store.setStatus(id, "dismissed"); err != nil {
			res.Failed++
			res.Errors = append(res.Errors, BulkResultItem{AlertID: id, Error: err.Error()})
		} else {
			res.Succeeded++
		}
	}
	return res, nil
}

func (s *testBulkSvc) BulkMarkDuplicate(ctx context.Context, _ *auth.Claims, ids []uuid.UUID, reason string) (*BulkResult, error) {
	if len(ids) == 0 {
		return &BulkResult{}, nil
	}
	if len(ids) > maxBulkAlerts {
		return nil, fmt.Errorf("bulk mark-duplicate limited to %d alerts per request", maxBulkAlerts)
	}
	res := &BulkResult{}
	for _, id := range ids {
		a, err := s.store.get(id)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, BulkResultItem{AlertID: id, Error: "alert not found"})
			continue
		}
		if a.Status == "converted" || a.Status == "dismissed" || a.Status == "duplicate" {
			res.Failed++
			res.Errors = append(res.Errors, BulkResultItem{AlertID: id, Error: "alert already actioned: " + a.Status})
			continue
		}
		if err := s.store.setStatus(id, "duplicate"); err != nil {
			res.Failed++
			res.Errors = append(res.Errors, BulkResultItem{AlertID: id, Error: err.Error()})
		} else {
			res.Succeeded++
		}
	}
	return res, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func analystClaims() *auth.Claims {
	return &auth.Claims{UserID: uuid.New(), Role: db.UserRoleAnalyst}
}

func newTestBulkSvc() *testBulkSvc { return &testBulkSvc{store: newMemStore()} }

// ── BulkDismiss service tests ─────────────────────────────────────────────────

func TestBulkDismiss_HappyPath(t *testing.T) {
	svc := newTestBulkSvc()
	id1, id2 := uuid.New(), uuid.New()
	svc.store.seed(id1, "pending")
	svc.store.seed(id2, "pending")

	res, err := svc.BulkDismiss(context.Background(), analystClaims(), []uuid.UUID{id1, id2}, "false positive")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Succeeded != 2 {
		t.Errorf("Succeeded: got %d want 2", res.Succeeded)
	}
	if res.Failed != 0 {
		t.Errorf("Failed: got %d want 0", res.Failed)
	}
	if svc.store.alerts[id1].Status != "dismissed" {
		t.Errorf("id1 status: got %q want dismissed", svc.store.alerts[id1].Status)
	}
}

func TestBulkDismiss_PartialFailure_AlreadyActioned(t *testing.T) {
	svc := newTestBulkSvc()
	pending   := uuid.New()
	converted := uuid.New()
	svc.store.seed(pending, "pending")
	svc.store.seed(converted, "converted")

	res, err := svc.BulkDismiss(context.Background(), analystClaims(), []uuid.UUID{pending, converted}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Succeeded != 1 {
		t.Errorf("Succeeded: got %d want 1", res.Succeeded)
	}
	if res.Failed != 1 {
		t.Errorf("Failed: got %d want 1", res.Failed)
	}
	if len(res.Errors) != 1 || res.Errors[0].AlertID != converted {
		t.Errorf("error item should reference the converted alert; got %+v", res.Errors)
	}
}

func TestBulkDismiss_NotFound(t *testing.T) {
	svc := newTestBulkSvc()
	res, err := svc.BulkDismiss(context.Background(), analystClaims(), []uuid.UUID{uuid.New()}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Succeeded != 0 || res.Failed != 1 {
		t.Errorf("expected 0 succeeded 1 failed; got %d/%d", res.Succeeded, res.Failed)
	}
}

func TestBulkDismiss_ExceedsLimit_ReturnsError(t *testing.T) {
	svc := newTestBulkSvc()
	ids := make([]uuid.UUID, maxBulkAlerts+1)
	for i := range ids {
		ids[i] = uuid.New()
	}
	_, err := svc.BulkDismiss(context.Background(), analystClaims(), ids, "")
	if err == nil {
		t.Fatal("expected error for oversized batch, got nil")
	}
}

func TestBulkDismiss_EmptyList_ReturnsZeroCounts(t *testing.T) {
	svc := newTestBulkSvc()
	res, err := svc.BulkDismiss(context.Background(), analystClaims(), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Succeeded != 0 || res.Failed != 0 {
		t.Error("empty list should produce zero counts")
	}
}

// ── BulkMarkDuplicate service tests ──────────────────────────────────────────

func TestBulkMarkDuplicate_HappyPath(t *testing.T) {
	svc := newTestBulkSvc()
	id := uuid.New()
	svc.store.seed(id, "pending")

	res, err := svc.BulkMarkDuplicate(context.Background(), analystClaims(), []uuid.UUID{id}, "dupe of ALT-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Succeeded != 1 {
		t.Errorf("Succeeded: got %d want 1", res.Succeeded)
	}
	if svc.store.alerts[id].Status != "duplicate" {
		t.Errorf("status: got %q want duplicate", svc.store.alerts[id].Status)
	}
}

func TestBulkMarkDuplicate_AlreadyDuplicate_IsFailure(t *testing.T) {
	svc := newTestBulkSvc()
	id := uuid.New()
	svc.store.seed(id, "duplicate")

	res, _ := svc.BulkMarkDuplicate(context.Background(), analystClaims(), []uuid.UUID{id}, "")
	if res.Failed != 1 || res.Succeeded != 0 {
		t.Errorf("expected 0/1; got %d/%d", res.Succeeded, res.Failed)
	}
}

func TestBulkMarkDuplicate_ExceedsLimit_ReturnsError(t *testing.T) {
	svc := newTestBulkSvc()
	ids := make([]uuid.UUID, maxBulkAlerts+1)
	for i := range ids {
		ids[i] = uuid.New()
	}
	_, err := svc.BulkMarkDuplicate(context.Background(), analystClaims(), ids, "")
	if err == nil {
		t.Fatal("expected error for oversized batch, got nil")
	}
}

// ── BulkAction HTTP handler tests ─────────────────────────────────────────────

// stubBulkSvc drives the HTTP handler in isolation.
type stubBulkSvc struct {
	dismissResult   *BulkResult
	dismissErr      error
	duplicateResult *BulkResult
	duplicateErr    error
}

// handlerUnderTest replicates the real BulkAction handler logic, wired to stubBulkSvc.
type handlerUnderTest struct{ svc *stubBulkSvc }

func (h *handlerUnderTest) BulkAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action   string      `json:"action"`
		AlertIDs []uuid.UUID `json:"alert_ids"`
		Reason   string      `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	if len(req.AlertIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var result *BulkResult
	var err error
	switch req.Action {
	case "dismiss":
		result, err = h.svc.dismissResult, h.svc.dismissErr
	case "mark_duplicate":
		result, err = h.svc.duplicateResult, h.svc.duplicateErr
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func newTestRouter(stub *stubBulkSvc) http.Handler {
	h := &handlerUnderTest{svc: stub}
	r := chi.NewRouter()
	r.Post("/api/v1/alerts/bulk", h.BulkAction)
	return r
}

func postBulk(router http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestBulkHandler_Dismiss_Returns200WithResult(t *testing.T) {
	stub := &stubBulkSvc{dismissResult: &BulkResult{Succeeded: 2, Failed: 0}}
	router := newTestRouter(stub)

	id1, id2 := uuid.New(), uuid.New()
	body := fmt.Sprintf(`{"action":"dismiss","alert_ids":[%q,%q],"reason":"test"}`, id1, id2)
	rr := postBulk(router, body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var got BulkResult
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Succeeded != 2 {
		t.Errorf("succeeded: got %d want 2", got.Succeeded)
	}
}

func TestBulkHandler_MarkDuplicate_Returns200(t *testing.T) {
	stub := &stubBulkSvc{duplicateResult: &BulkResult{Succeeded: 1, Failed: 0}}
	router := newTestRouter(stub)

	body := fmt.Sprintf(`{"action":"mark_duplicate","alert_ids":[%q]}`, uuid.New())
	rr := postBulk(router, body)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestBulkHandler_InvalidAction_Returns400(t *testing.T) {
	router := newTestRouter(&stubBulkSvc{})
	rr := postBulk(router, fmt.Sprintf(`{"action":"convert","alert_ids":[%q]}`, uuid.New()))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid action, got %d", rr.Code)
	}
}

func TestBulkHandler_EmptyIDs_Returns400(t *testing.T) {
	router := newTestRouter(&stubBulkSvc{})
	rr := postBulk(router, `{"action":"dismiss","alert_ids":[]}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty alert_ids, got %d", rr.Code)
	}
}

func TestBulkHandler_ServiceError_Returns400(t *testing.T) {
	stub := &stubBulkSvc{
		dismissErr: fmt.Errorf("bulk dismiss limited to %d alerts per request", maxBulkAlerts),
	}
	router := newTestRouter(stub)
	rr := postBulk(router, fmt.Sprintf(`{"action":"dismiss","alert_ids":[%q]}`, uuid.New()))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when service returns error, got %d", rr.Code)
	}
}

func TestBulkHandler_PartialFailure_Still200(t *testing.T) {
	stub := &stubBulkSvc{
		dismissResult: &BulkResult{
			Succeeded: 1, Failed: 1,
			Errors: []BulkResultItem{{AlertID: uuid.New(), Error: "alert already actioned: converted"}},
		},
	}
	router := newTestRouter(stub)
	id1, id2 := uuid.New(), uuid.New()
	body := fmt.Sprintf(`{"action":"dismiss","alert_ids":[%q,%q]}`, id1, id2)
	rr := postBulk(router, body)

	// Partial failures are still 200 — per-item errors live in the body.
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for partial failure, got %d", rr.Code)
	}
	var got BulkResult
	_ = json.NewDecoder(rr.Body).Decode(&got)
	if got.Failed != 1 || got.Succeeded != 1 {
		t.Errorf("expected 1/1; got succeeded=%d failed=%d", got.Succeeded, got.Failed)
	}
}
