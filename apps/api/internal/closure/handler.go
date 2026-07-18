package closure

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

// GET /api/v1/cases/:id/closure/checklist
func (h *Handler) Checklist(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	result, err := h.svc.Checklist(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "CHECKLIST_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GET /api/v1/cases/:id/closure
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	cl, err := h.svc.Get(r.Context(), caseID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// No closure record yet — return 200 with not_started so clients
			// don't treat an uninitiated closure as an error.
			response.JSON(w, http.StatusOK, ClosureResponse{
				CaseID: caseID.String(),
				Status: "not_started",
			})
			return
		}
		response.Err(w, http.StatusInternalServerError, "GET_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, cl)
}

// POST /api/v1/cases/:id/closure
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req CreateClosureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	cl, err := h.svc.Create(r.Context(), claims, caseID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, cl)
}

// PATCH /api/v1/cases/:id/closure/:closureId
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	caseID, closureID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}

	var req UpdateClosureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	cl, err := h.svc.Update(r.Context(), claims, caseID, closureID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cl)
}

// POST /api/v1/cases/:id/closure/:closureId/submit
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	caseID, closureID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	cl, err := h.svc.Submit(r.Context(), claims, caseID, closureID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cl)
}

// POST /api/v1/cases/:id/closure/:closureId/approve
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	caseID, closureID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	cl, err := h.svc.Approve(r.Context(), claims, caseID, closureID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cl)
}

// POST /api/v1/cases/:id/closure/:closureId/reject
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	caseID, closureID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	var req ReviewClosureRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	claims := auth.ClaimsFromCtx(r.Context())
	cl, err := h.svc.Reject(r.Context(), claims, caseID, closureID, req.Reason)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cl)
}

// POST /api/v1/cases/:id/closure/:closureId/revise
func (h *Handler) Revise(w http.ResponseWriter, r *http.Request) {
	caseID, closureID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	cl, err := h.svc.Revise(r.Context(), claims, caseID, closureID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, cl)
}

// POST /api/v1/cases/:id/reopen
func (h *Handler) Reopen(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	var req ReopenRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Reopen(r.Context(), claims, caseID, req.Reason); err != nil {
		h.writeError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"reopened": true})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) parseIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return uuid.Nil, uuid.Nil, false
	}
	closureID, err := uuid.Parse(chi.URLParam(r, "closureId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid closure id")
		return uuid.Nil, uuid.Nil, false
	}
	return caseID, closureID, true
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrForbidden):
		response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, ErrBadRequest), errors.Is(err, ErrSummaryTooShort):
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, ErrChecklistFailed):
		response.Err(w, http.StatusUnprocessableEntity, "CHECKLIST_FAILED", err.Error())
	case errors.Is(err, ErrSARRejectionRequired):
		response.Err(w, http.StatusUnprocessableEntity, "SAR_REJECTION_REQUIRED", err.Error())
	case errors.Is(err, ErrSamePersonApproval):
		response.Err(w, http.StatusConflict, "SAME_PERSON_APPROVAL", err.Error())
	case errors.Is(err, ErrInvalidTransition):
		response.Err(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
