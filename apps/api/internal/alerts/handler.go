package alerts

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

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

// ── Webhook ingest — no JWT, HMAC auth ───────────────────────────────────────

// IngestWebhook handles inbound payloads from TMS systems.
// URL: POST /api/v1/alerts/ingest/{sourceID}
// Auth: X-OpsOwl-Signature: sha256=<hex>   (HMAC-SHA256 over body, keyed with source secret)
func (h *Handler) IngestWebhook(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_SOURCE_ID", "invalid source id")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20)) // 4 MB cap
	if err != nil {
		response.Err(w, http.StatusBadRequest, "READ_ERROR", "failed to read request body")
		return
	}

	// Fetch source to get the signing secret
	src, err := h.svc.GetSource(r.Context(), sourceID)
	if err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			response.Err(w, http.StatusNotFound, "SOURCE_NOT_FOUND", "alert source not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load source")
		return
	}

	// Skip signature check if no secret has been set (allows easy local dev)
	sig := r.Header.Get("X-OpsOwl-Signature")
	if src.WebhookSecret != "" && sig != "" {
		if !VerifyWebhookSignature(src.WebhookSecret, body, sig) {
			response.Err(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "webhook signature mismatch")
			return
		}
	} else if src.WebhookSecret != "" && sig == "" {
		response.Err(w, http.StatusUnauthorized, "MISSING_SIGNATURE", "X-OpsOwl-Signature header required")
		return
	}

	result, err := h.svc.IngestAlert(r.Context(), sourceID, body)
	if err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			response.Err(w, http.StatusNotFound, "SOURCE_NOT_FOUND", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INGEST_ERROR", err.Error())
		return
	}

	status := http.StatusCreated
	if result.Status == "deduplicated" {
		status = http.StatusOK
	}
	response.JSON(w, status, result)
}

// ManualIngest handles manually submitted alert payloads from the frontend.
// URL: POST /api/v1/alerts/manual/{sourceID}
// Auth: JWT (analyst+)
func (h *Handler) ManualIngest(w http.ResponseWriter, r *http.Request) {
	sourceID, err := uuid.Parse(chi.URLParam(r, "sourceID"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_SOURCE_ID", "invalid source id")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "READ_ERROR", "failed to read request body")
		return
	}
	result, err := h.svc.IngestAlert(r.Context(), sourceID, body)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INGEST_ERROR", err.Error())
		return
	}
	status := http.StatusCreated
	if result.Status == "deduplicated" {
		status = http.StatusOK
	}
	response.JSON(w, status, result)
}

// ── Alert queue ───────────────────────────────────────────────────────────────

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := AlertListFilters{Limit: 25}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			f.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}
	if v := q.Get("status"); v != "" {
		f.Status = v
	}
	if v := q.Get("alert_type"); v != "" {
		f.AlertType = strings.ToUpper(v)
	}
	if v := q.Get("risk_level"); v != "" {
		f.RiskLevel = strings.ToLower(v)
	}
	if v := q.Get("source_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.SourceID = &id
		}
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list alerts")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"alerts": items,
		"total":  total,
		"limit":  f.Limit,
		"offset": f.Offset,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid alert id")
		return
	}
	alert, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get alert")
		return
	}
	response.JSON(w, http.StatusOK, alert)
}

func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid alert id")
		return
	}
	var req ConvertAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	result, err := h.svc.ConvertToCase(r.Context(), claims, id, req)
	if err != nil {
		if errors.Is(err, ErrAlreadyConverted) {
			response.Err(w, http.StatusConflict, "ALREADY_CONVERTED", err.Error())
			return
		}
		if errors.Is(err, ErrAlreadyDismissed) {
			response.Err(w, http.StatusConflict, "ALREADY_DISMISSED", err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to convert alert")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Dismiss(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid alert id")
		return
	}
	var req DismissAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.svc.DismissAlert(r.Context(), claims, id, req); err != nil {
		if errors.Is(err, ErrAlreadyDismissed) {
			response.Err(w, http.StatusConflict, "ALREADY_DISMISSED", err.Error())
			return
		}
		if errors.Is(err, ErrAlreadyConverted) {
			response.Err(w, http.StatusConflict, "ALREADY_CONVERTED", err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to dismiss alert")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// ── Alert Sources ─────────────────────────────────────────────────────────────

func (h *Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.svc.ListSources(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list sources")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"sources": sources})
}

func (h *Handler) CreateSource(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if req.SourceSystem == "" {
		req.SourceSystem = "generic"
	}
	if req.SourceType == "" {
		req.SourceType = "webhook"
	}
	src, err := h.svc.CreateSource(r.Context(), claims, req)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create source")
		return
	}
	response.JSON(w, http.StatusCreated, src)
}

func (h *Handler) GetSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source id")
		return
	}
	src, err := h.svc.GetSource(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get source")
		return
	}
	// Never expose the secret in GET
	src.WebhookSecret = ""
	response.JSON(w, http.StatusOK, src)
}

func (h *Handler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source id")
		return
	}
	var req UpdateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	src, err := h.svc.UpdateSource(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update source")
		return
	}
	src.WebhookSecret = ""
	response.JSON(w, http.StatusOK, src)
}

func (h *Handler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid source id")
		return
	}
	if err := h.svc.DeleteSource(r.Context(), id); err != nil {
		if errors.Is(err, ErrSourceNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "source not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete source")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Alert Rules ───────────────────────────────────────────────────────────────

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.svc.ListRules(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list rules")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"rules": rules})
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if req.AlertType == "" {
		req.AlertType = "*"
	}
	rule, err := h.svc.CreateRule(r.Context(), claims, req)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create rule")
		return
	}
	response.JSON(w, http.StatusCreated, rule)
}

func (h *Handler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid rule id")
		return
	}
	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	rule, err := h.svc.UpdateRule(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "rule not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update rule")
		return
	}
	response.JSON(w, http.StatusOK, rule)
}

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid rule id")
		return
	}
	if err := h.svc.DeleteRule(r.Context(), id); err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "rule not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete rule")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
