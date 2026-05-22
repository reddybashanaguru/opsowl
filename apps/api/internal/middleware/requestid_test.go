package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID_GeneratesIDWhenAbsent(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromCtx(r.Context())
		if id == "" {
			t.Error("request ID missing from context")
		}
		if _, err := uuid.Parse(id); err != nil {
			t.Errorf("request ID is not a valid UUID: %q", id)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id not set in response header")
	}
}

func TestRequestID_ReusesCallerID(t *testing.T) {
	const callerID = "my-upstream-trace-id"
	var captured string

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = RequestIDFromCtx(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", callerID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured != callerID {
		t.Errorf("want %q, got %q", callerID, captured)
	}
	if rr.Header().Get("X-Request-Id") != callerID {
		t.Errorf("response header: want %q, got %q", callerID, rr.Header().Get("X-Request-Id"))
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	ids := make(map[string]struct{})
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids[RequestIDFromCtx(r.Context())] = struct{}{}
	}))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	if len(ids) != 100 {
		t.Errorf("expected 100 unique IDs, got %d (collisions detected)", len(ids))
	}
}

func TestRequestIDFromCtx_NilContextReturnsEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No RequestID middleware applied — context has no ID
	id := RequestIDFromCtx(req.Context())
	if id != "" {
		t.Errorf("expected empty string for missing request ID, got %q", id)
	}
}
