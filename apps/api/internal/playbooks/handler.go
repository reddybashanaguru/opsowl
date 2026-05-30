package playbooks

import (
	"encoding/json"
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pbs, err := h.svc.List(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if pbs == nil {
		pbs = []PlaybookResponse{}
	}
	response.JSON(w, http.StatusOK, pbs)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreatePlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	pb, err := h.svc.Create(r.Context(), claims, req)
	if err != nil {
		switch err {
		case ErrNameRequired, ErrCaseTypeRequired:
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusCreated, pb)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid playbook id")
		return
	}
	pb, err := h.svc.Get(r.Context(), id)
	if err == ErrNotFound {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "playbook not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, pb)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid playbook id")
		return
	}
	var req UpdatePlaybookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	pb, err := h.svc.Update(r.Context(), claims, id, req)
	switch err {
	case nil:
		response.JSON(w, http.StatusOK, pb)
	case ErrNotFound:
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "playbook not found")
	case ErrForbidden:
		response.Err(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func (h *Handler) ReplaceSteps(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid playbook id")
		return
	}
	var steps []CreateStepRequest
	if err := json.NewDecoder(r.Body).Decode(&steps); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON")
		return
	}
	pb, err := h.svc.ReplaceSteps(r.Context(), claims, id, steps)
	switch err {
	case nil:
		response.JSON(w, http.StatusOK, pb)
	case ErrNotFound:
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "playbook not found")
	case ErrForbidden:
		response.Err(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid playbook id")
		return
	}
	err = h.svc.Delete(r.Context(), claims, id)
	switch err {
	case nil:
		w.WriteHeader(http.StatusNoContent)
	case ErrNotFound:
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "playbook not found")
	case ErrForbidden:
		response.Err(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
