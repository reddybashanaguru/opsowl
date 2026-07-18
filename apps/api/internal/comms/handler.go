package comms

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createCommLogRequest struct {
	Channel       string `json:"channel"`
	Direction     string `json:"direction"`
	ContactName   string `json:"contact_name"`
	ContactDetail string `json:"contact_detail"`
	Subject       string `json:"subject"`
	Body          string `json:"body"`
	Outcome       string `json:"outcome"`
	OccurredAt    string `json:"occurred_at"`
}

// GET /api/v1/cases/{caseId}/comms
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	limit := int32(50)
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.ParseInt(v, 10, 32); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	logs, err := h.svc.List(r.Context(), caseID, limit)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, logs)
}

// POST /api/v1/cases/{caseId}/comms
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req createCommLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Body == "" || req.ContactName == "" || req.Channel == "" || req.Direction == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_FIELDS", "channel, direction, contact_name, body are required")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())

	occurredAt := time.Now().UTC()
	if req.OccurredAt != "" {
		if t, err := time.Parse(time.RFC3339, req.OccurredAt); err == nil {
			occurredAt = t
		}
	}

	log, err := h.svc.Create(r.Context(), CreateParams{
		CaseID:        caseID,
		LoggedBy:      claims.UserID,
		Channel:       db.CommChannel(req.Channel),
		Direction:     db.CommDirection(req.Direction),
		ContactName:   req.ContactName,
		ContactDetail: req.ContactDetail,
		Subject:       req.Subject,
		Body:          req.Body,
		Outcome:       req.Outcome,
		OccurredAt:    occurredAt,
	})
	if err != nil {
		if errors.Is(err, ErrCaseClosed) {
			response.Err(w, http.StatusConflict, "CASE_CLOSED", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, log)
}

// DELETE /api/v1/cases/{caseId}/comms/{commId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	commID, err := uuid.Parse(chi.URLParam(r, "commId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid comm id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Delete(r.Context(), commID, claims.UserID); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}
