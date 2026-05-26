package inbound

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ─── Inbound event endpoint (HMAC-authenticated, no JWT required) ─────────────

// ReceiveEvent is the webhook endpoint external systems call to push events.
//
// Required headers:
//   X-OpsOwl-Source:          <source-slug>        (identifies the caller)
//   X-OpsOwl-Signature:       sha256=<hmac-hex>     (proves authenticity)
//   X-OpsOwl-Idempotency-Key: <unique-key>          (prevents duplicate cases)
//
// Body: { "event_type": "fraud_alert", "payload": { ... } }
func (h *Handler) ReceiveEvent(w http.ResponseWriter, r *http.Request) {
	sourceSlug := r.Header.Get("X-OpsOwl-Source")
	signature := r.Header.Get("X-OpsOwl-Signature")
	idempotencyKey := r.Header.Get("X-OpsOwl-Idempotency-Key")

	if sourceSlug == "" || signature == "" || idempotencyKey == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_HEADERS",
			"X-OpsOwl-Source, X-OpsOwl-Signature, and X-OpsOwl-Idempotency-Key headers are required")
		return
	}

	// Read body — needed for HMAC validation and parsing.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "could not read request body")
		return
	}

	eventID, isDuplicate, err := h.svc.Receive(r.Context(), sourceSlug, signature, idempotencyKey, body)
	switch {
	case errors.Is(err, ErrSourceNotFound):
		response.Err(w, http.StatusUnauthorized, "UNKNOWN_SOURCE", "unknown source identifier")
	case errors.Is(err, ErrSourceInactive):
		response.Err(w, http.StatusForbidden, "SOURCE_DISABLED", "this inbound source is disabled")
	case errors.Is(err, ErrInvalidSignature):
		response.Err(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "HMAC signature verification failed")
	case errors.Is(err, ErrDuplicateEvent):
		// Return 200 (not 409) — the caller already sent this, we just acknowledge it.
		response.JSON(w, http.StatusOK, map[string]any{
			"status":   "duplicate",
			"message":  "event already received",
			"event_id": nil,
		})
	case err != nil:
		response.Err(w, http.StatusBadRequest, "INVALID_EVENT", err.Error())
	default:
		if isDuplicate {
			response.JSON(w, http.StatusOK, map[string]any{
				"status":   "duplicate",
				"message":  "event already received",
			})
			return
		}
		response.JSON(w, http.StatusAccepted, map[string]any{
			"status":   "queued",
			"event_id": eventID,
			"message":  "event accepted and queued for processing",
		})
	}
}

// ─── Admin: Source management ─────────────────────────────────────────────────

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.svc.ListSources(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list sources")
		return
	}
	if sources == nil {
		sources = []*Source{}
	}
	response.JSON(w, http.StatusOK, sources)
}

func (h *Handler) CreateSource(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	src, err := h.svc.CreateSource(r.Context(), claims.UserID, req)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, src)
}

func (h *Handler) GetSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source ID")
		return
	}
	src, err := h.svc.GetSource(r.Context(), id)
	if errors.Is(err, ErrSourceNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not get source")
		return
	}
	response.JSON(w, http.StatusOK, src)
}

func (h *Handler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source ID")
		return
	}
	var req UpdateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	src, err := h.svc.UpdateSource(r.Context(), id, req)
	if errors.Is(err, ErrSourceNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update source")
		return
	}
	response.JSON(w, http.StatusOK, src)
}

func (h *Handler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source ID")
		return
	}
	if err := h.svc.DeleteSource(r.Context(), id); errors.Is(err, ErrSourceNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RegenerateSecret(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source ID")
		return
	}
	secret, err := h.svc.RegenerateSecret(r.Context(), id)
	if errors.Is(err, ErrSourceNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not regenerate secret")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"signing_secret": secret})
}

// ─── Admin: Rule management ───────────────────────────────────────────────────

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source ID")
		return
	}
	rules, err := h.svc.ListRules(r.Context(), sourceID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list rules")
		return
	}
	if rules == nil {
		rules = []*Rule{}
	}
	response.JSON(w, http.StatusOK, rules)
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source ID")
		return
	}
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	rule, err := h.svc.CreateRule(r.Context(), sourceID, claims.UserID, req)
	if err != nil {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid rule ID")
		return
	}
	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	rule, err := h.svc.UpdateRule(r.Context(), ruleID, req)
	if errors.Is(err, ErrRuleNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "rule not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update rule")
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(chi.URLParam(r, "ruleID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid rule ID")
		return
	}
	if err := h.svc.DeleteRule(r.Context(), ruleID); errors.Is(err, ErrRuleNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "rule not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Admin: Event log ─────────────────────────────────────────────────────────

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	var sourceID *uuid.UUID
	if s := q.Get("source_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			sourceID = &id
		}
	}
	var status *string
	if s := q.Get("status"); s != "" {
		status = &s
	}
	var cursorID *uuid.UUID
	if s := q.Get("cursor"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			cursorID = &id
		}
	}
	page, err := h.svc.ListEvents(r.Context(), limit, sourceID, status, cursorID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list events")
		return
	}
	response.JSON(w, http.StatusOK, page)
}
