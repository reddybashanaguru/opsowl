package webhooks

import (
	"encoding/json"
	"errors"
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

// GET /api/v1/webhooks
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	hooks, err := h.svc.List(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list webhooks")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"webhooks": hooks})
}

// POST /api/v1/webhooks
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	// Auto-generate secret if omitted
	if req.SigningSecret == "" {
		secret, err := GenerateSecret()
		if err != nil {
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not generate signing secret")
			return
		}
		req.SigningSecret = secret
	}
	claims := auth.ClaimsFromCtx(r.Context())
	hook, err := h.svc.Create(r.Context(), req, claimsActor(claims))
	switch {
	case errors.Is(err, ErrInvalidURL):
		response.Err(w, http.StatusBadRequest, "INVALID_URL", ErrInvalidURL.Error())
	case errors.Is(err, ErrInvalidSecret):
		response.Err(w, http.StatusBadRequest, "INVALID_SECRET", ErrInvalidSecret.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create webhook")
	default:
		response.JSON(w, http.StatusCreated, hook)
	}
}

// GET /api/v1/webhooks/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid webhook ID")
		return
	}
	hook, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "webhook not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not fetch webhook")
		return
	}
	response.JSON(w, http.StatusOK, hook)
}

// PUT /api/v1/webhooks/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid webhook ID")
		return
	}
	var req UpdateWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	hook, err := h.svc.Update(r.Context(), id, req)
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "webhook not found")
	case errors.Is(err, ErrInvalidURL):
		response.Err(w, http.StatusBadRequest, "INVALID_URL", ErrInvalidURL.Error())
	case errors.Is(err, ErrInvalidSecret):
		response.Err(w, http.StatusBadRequest, "INVALID_SECRET", ErrInvalidSecret.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update webhook")
	default:
		response.JSON(w, http.StatusOK, hook)
	}
}

// DELETE /api/v1/webhooks/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid webhook ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "webhook not found")
		return
	} else if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete webhook")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/webhooks/{id}/deliveries
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid webhook ID")
		return
	}
	limit := int32(25)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	page, err := h.svc.ListDeliveries(r.Context(), id, limit, offset)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list deliveries")
		return
	}
	response.JSON(w, http.StatusOK, page)
}

// POST /api/v1/webhooks/{id}/deliveries/{deliveryId}/redeliver
func (h *Handler) Redeliver(w http.ResponseWriter, r *http.Request) {
	deliveryID, err := uuid.Parse(chi.URLParam(r, "deliveryId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid delivery ID")
		return
	}
	if err := h.svc.Redeliver(r.Context(), deliveryID); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not redeliver")
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

// GET /api/v1/webhooks/event-types
func (h *Handler) ListEventTypes(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]any{"event_types": KnownEventTypes})
}

func claimsActor(claims *auth.Claims) *uuid.UUID {
	if claims == nil {
		return nil
	}
	id := claims.UserID
	return &id
}
