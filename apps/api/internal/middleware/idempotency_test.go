package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// ── bodyRecorder unit tests ──────────────────────────────────────────────────

func TestBodyRecorder_CapturesStatusAndBody(t *testing.T) {
	rr := httptest.NewRecorder()
	rec := &bodyRecorder{ResponseWriter: rr, status: http.StatusOK}

	rec.WriteHeader(http.StatusCreated)
	rec.Write([]byte(`{"id":"abc"}`)) //nolint:errcheck

	if rec.status != http.StatusCreated {
		t.Errorf("want status 201, got %d", rec.status)
	}
	if rec.body.String() != `{"id":"abc"}` {
		t.Errorf("want body %q, got %q", `{"id":"abc"}`, rec.body.String())
	}
	// Also written to underlying ResponseWriter
	if rr.Code != http.StatusCreated {
		t.Errorf("underlying recorder status: want 201, got %d", rr.Code)
	}
	if rr.Body.String() != `{"id":"abc"}` {
		t.Errorf("underlying recorder body: want %q, got %q", `{"id":"abc"}`, rr.Body.String())
	}
}

func TestBodyRecorder_DefaultStatus200(t *testing.T) {
	rr := httptest.NewRecorder()
	rec := &bodyRecorder{ResponseWriter: rr, status: http.StatusOK}
	rec.Write([]byte("ok")) //nolint:errcheck
	if rec.status != http.StatusOK {
		t.Errorf("default status should be 200, got %d", rec.status)
	}
}

// ── Idempotency middleware integration tests (SQL-free) ──────────────────────
// These tests verify the middleware's branching logic without a real DB by
// using a nil *sql.DB and testing only the no-key / non-POST paths.

func newIdempotencyHandler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body) //nolint:errcheck
	})
}

func TestIdempotency_PassthroughForGET(t *testing.T) {
	mw := Idempotency(nil, zap.NewNop())
	handler := mw(newIdempotencyHandler(http.StatusOK, `{"ok":true}`))

	req := httptest.NewRequest(http.MethodGet, "/cases", nil)
	req.Header.Set("Idempotency-Key", "key-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// GET should pass straight through
	if rr.Code != http.StatusOK {
		t.Errorf("GET with Idempotency-Key: want 200, got %d", rr.Code)
	}
}

func TestIdempotency_PassthroughForPUT(t *testing.T) {
	mw := Idempotency(nil, zap.NewNop())
	handler := mw(newIdempotencyHandler(http.StatusOK, `{"ok":true}`))

	req := httptest.NewRequest(http.MethodPut, "/cases/abc", bytes.NewBufferString(`{}`))
	req.Header.Set("Idempotency-Key", "key-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("PUT with Idempotency-Key: want 200, got %d", rr.Code)
	}
}

func TestIdempotency_NoKeyPassesThroughPOST(t *testing.T) {
	mw := Idempotency(nil, zap.NewNop())
	handler := mw(newIdempotencyHandler(http.StatusCreated, `{"id":"xyz"}`))

	req := httptest.NewRequest(http.MethodPost, "/cases", bytes.NewBufferString(`{}`))
	// No Idempotency-Key header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// No key → middleware transparent, handler response returned
	if rr.Code != http.StatusCreated {
		t.Errorf("POST without key: want 201, got %d", rr.Code)
	}
	if rr.Body.String() != `{"id":"xyz"}` {
		t.Errorf("body mismatch: %q", rr.Body.String())
	}
}

func TestIdempotency_NoAuthContextPassesThroughPOST(t *testing.T) {
	mw := Idempotency(nil, zap.NewNop())
	handler := mw(newIdempotencyHandler(http.StatusCreated, `{"id":"xyz"}`))

	req := httptest.NewRequest(http.MethodPost, "/cases", bytes.NewBufferString(`{}`))
	req.Header.Set("Idempotency-Key", "key-abc")
	// No auth claims in context → middleware passes through
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("POST with key but no auth: want 201, got %d", rr.Code)
	}
}
