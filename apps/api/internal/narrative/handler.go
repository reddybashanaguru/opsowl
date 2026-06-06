package narrative

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

// GET /api/v1/cases/:id/narratives
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	rows, err := h.svc.List(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rows)
}

// GET /api/v1/cases/:id/narratives/:narrativeId
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	caseID, narrativeID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	row, err := h.svc.Get(r.Context(), caseID, narrativeID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, row)
}

// POST /api/v1/cases/:id/narratives/generate
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req GenerateRequest
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

	claims := auth.ClaimsFromCtx(r.Context())
	row, err := h.svc.Generate(r.Context(), claims, caseID, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, row)
}

// PATCH /api/v1/cases/:id/narratives/:narrativeId
func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	caseID, narrativeID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}

	var req EditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	row, err := h.svc.Edit(r.Context(), claims, caseID, narrativeID, req)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, row)
}

// POST /api/v1/cases/:id/narratives/:narrativeId/approve
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	caseID, narrativeID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	row, err := h.svc.Approve(r.Context(), claims, caseID, narrativeID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, row)
}

// POST /api/v1/cases/:id/narratives/:narrativeId/reject
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	caseID, narrativeID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}

	var req ReviewRequest
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

	claims := auth.ClaimsFromCtx(r.Context())
	row, err := h.svc.Reject(r.Context(), claims, caseID, narrativeID, req.Reason)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, row)
}

// DELETE /api/v1/cases/:id/narratives/:narrativeId
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	caseID, narrativeID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Delete(r.Context(), claims, caseID, narrativeID); err != nil {
		h.writeErr(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) parseIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return uuid.Nil, uuid.Nil, false
	}
	narrativeID, err := uuid.Parse(chi.URLParam(r, "narrativeId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid narrative id")
		return uuid.Nil, uuid.Nil, false
	}
	return caseID, narrativeID, true
}

func (h *Handler) writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrForbidden):
		response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, ErrBadRequest):
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, ErrAlreadyApproved):
		response.Err(w, http.StatusConflict, "ALREADY_APPROVED", err.Error())
	case errors.Is(err, ErrInvalidTransition):
		response.Err(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
