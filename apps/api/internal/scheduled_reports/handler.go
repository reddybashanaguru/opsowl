package scheduled_reports

import (
	"encoding/json"
	"errors"
	"net/http"

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

// GET /api/v1/report-schedules
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	schedules, err := h.svc.List(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list schedules")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"schedules": schedules})
}

// POST /api/v1/report-schedules
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	sched, err := h.svc.Create(r.Context(), req, claims)
	switch {
	case errors.Is(err, ErrNoRecipients):
		response.Err(w, http.StatusBadRequest, "NO_RECIPIENTS", err.Error())
	case errors.Is(err, ErrInvalidFrequency):
		response.Err(w, http.StatusBadRequest, "INVALID_FREQUENCY", err.Error())
	case errors.Is(err, ErrInvalidDayOfWeek):
		response.Err(w, http.StatusBadRequest, "INVALID_DAY_OF_WEEK", err.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create schedule")
	default:
		response.JSON(w, http.StatusCreated, sched)
	}
}

// GET /api/v1/report-schedules/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID")
		return
	}
	sched, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not fetch schedule")
		return
	}
	response.JSON(w, http.StatusOK, sched)
}

// PUT /api/v1/report-schedules/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID")
		return
	}
	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	sched, err := h.svc.Update(r.Context(), id, req)
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
	case errors.Is(err, ErrNoRecipients):
		response.Err(w, http.StatusBadRequest, "NO_RECIPIENTS", err.Error())
	case errors.Is(err, ErrInvalidFrequency):
		response.Err(w, http.StatusBadRequest, "INVALID_FREQUENCY", err.Error())
	case errors.Is(err, ErrInvalidDayOfWeek):
		response.Err(w, http.StatusBadRequest, "INVALID_DAY_OF_WEEK", err.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update schedule")
	default:
		response.JSON(w, http.StatusOK, sched)
	}
}

// DELETE /api/v1/report-schedules/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
		return
	} else if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete schedule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/report-schedules/{id}/send-now
func (h *Handler) SendNow(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid schedule ID")
		return
	}
	if err := h.svc.SendNow(r.Context(), id); errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "schedule not found")
		return
	} else if err != nil {
		response.Err(w, http.StatusInternalServerError, "SEND_ERROR", "failed to send report: "+err.Error())
		return
	}
	response.JSON(w, http.StatusAccepted, map[string]string{"status": "sent"})
}
