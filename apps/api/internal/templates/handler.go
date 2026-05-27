package templates

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

// GET /api/v1/templates?active=true
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"
	tmplts, err := h.svc.List(r.Context(), activeOnly)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not list templates")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"templates": tmplts})
}

// POST /api/v1/templates
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	tmpl, err := h.svc.Create(r.Context(), req, claims)
	switch {
	case errors.Is(err, ErrNameRequired):
		response.Err(w, http.StatusBadRequest, "NAME_REQUIRED", err.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create template")
	default:
		response.JSON(w, http.StatusCreated, tmpl)
	}
}

// GET /api/v1/templates/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
		return
	}
	tmpl, err := h.svc.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "template not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not fetch template")
		return
	}
	response.JSON(w, http.StatusOK, tmpl)
}

// PUT /api/v1/templates/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
		return
	}
	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return
	}
	tmpl, err := h.svc.Update(r.Context(), id, req)
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "template not found")
	case errors.Is(err, ErrNameRequired):
		response.Err(w, http.StatusBadRequest, "NAME_REQUIRED", err.Error())
	case err != nil:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not update template")
	default:
		response.JSON(w, http.StatusOK, tmpl)
	}
}

// DELETE /api/v1/templates/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
		return
	}
	if err := h.svc.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "template not found")
		return
	} else if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not delete template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/templates/{id}/apply
// Body: {"case_id": "<uuid>"}
func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid template ID")
		return
	}
	var body struct {
		CaseID uuid.UUID `json:"case_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CaseID == uuid.Nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", "case_id is required")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Apply(r.Context(), id, body.CaseID, claims); errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "template not found")
		return
	} else if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "could not apply template")
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "applied"})
}
