package findings

import (
	"encoding/json"
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

// GET /api/v1/cases/:id/findings
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	q := r.URL.Query()
	statusFilter := q.Get("status")
	severityFilter := q.Get("severity")
	cursor := q.Get("cursor")
	limit := 25
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	findings, meta, err := h.svc.List(r.Context(), caseID, statusFilter, severityFilter, limit, cursor)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}

	response.JSONWithMeta(w, http.StatusOK, findings, meta)
}

// POST /api/v1/cases/:id/findings
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req CreateFindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	finding, err := h.svc.Create(r.Context(), claims, caseID, req)
	if err != nil {
		switch err {
		case ErrBadRequest:
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "CREATE_ERROR", err.Error())
		}
		return
	}

	response.JSON(w, http.StatusCreated, finding)
}

// GET /api/v1/cases/:id/findings/:findingId
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	findingID, err := uuid.Parse(chi.URLParam(r, "findingId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid finding id")
		return
	}

	finding, err := h.svc.Get(r.Context(), caseID, findingID)
	if err != nil {
		if err == ErrNotFound {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "finding not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "GET_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, finding)
}

// PATCH /api/v1/cases/:id/findings/:findingId
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	findingID, err := uuid.Parse(chi.URLParam(r, "findingId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid finding id")
		return
	}

	var req UpdateFindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	finding, err := h.svc.Update(r.Context(), claims, caseID, findingID, req)
	if err != nil {
		switch err {
		case ErrNotFound:
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "finding not found")
		case ErrForbidden:
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		case ErrBadRequest:
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "UPDATE_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusOK, finding)
}

// DELETE /api/v1/cases/:id/findings/:findingId
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	findingID, err := uuid.Parse(chi.URLParam(r, "findingId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid finding id")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.Delete(r.Context(), claims, caseID, findingID); err != nil {
		switch err {
		case ErrNotFound:
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "finding not found")
		case ErrForbidden:
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "DELETE_ERROR", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/cases/:id/findings/:findingId/submit
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	finding, err := h.svc.Submit(r.Context(), claims, caseID, findingID)
	h.writeTransitionResponse(w, finding, err)
}

// POST /api/v1/cases/:id/findings/:findingId/approve
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	finding, err := h.svc.Approve(r.Context(), claims, caseID, findingID)
	h.writeTransitionResponse(w, finding, err)
}

// POST /api/v1/cases/:id/findings/:findingId/reject
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	var req ReviewFindingRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	claims := auth.ClaimsFromCtx(r.Context())
	finding, err := h.svc.Reject(r.Context(), claims, caseID, findingID, req.Reason)
	h.writeTransitionResponse(w, finding, err)
}

// POST /api/v1/cases/:id/findings/:findingId/contest
func (h *Handler) Contest(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	var req ReviewFindingRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	claims := auth.ClaimsFromCtx(r.Context())
	finding, err := h.svc.Contest(r.Context(), claims, caseID, findingID, req.Reason)
	h.writeTransitionResponse(w, finding, err)
}

// GET /api/v1/cases/:id/findings/summary
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	summary, err := h.svc.Summary(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "SUMMARY_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, summary)
}

// POST /api/v1/cases/:id/findings/:findingId/evidence
func (h *Handler) LinkEvidence(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}

	var req LinkEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	link, err := h.svc.LinkEvidence(r.Context(), claims, caseID, findingID, req)
	if err != nil {
		switch err {
		case ErrNotFound:
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "finding not found")
		case ErrBadRequest:
			response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "LINK_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusCreated, link)
}

// GET /api/v1/cases/:id/findings/:findingId/evidence
func (h *Handler) ListEvidence(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}

	links, err := h.svc.ListEvidence(r.Context(), caseID, findingID)
	if err != nil {
		if err == ErrNotFound {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "finding not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "LIST_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, links)
}

// DELETE /api/v1/cases/:id/findings/:findingId/evidence/:linkId
func (h *Handler) RemoveEvidence(w http.ResponseWriter, r *http.Request) {
	caseID, findingID, ok := h.parseIDs(w, r)
	if !ok {
		return
	}
	linkIDStr := chi.URLParam(r, "linkId")

	claims := auth.ClaimsFromCtx(r.Context())
	if err := h.svc.RemoveEvidence(r.Context(), claims, caseID, findingID, linkIDStr); err != nil {
		switch err {
		case ErrNotFound:
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "evidence link not found")
		case ErrForbidden:
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "REMOVE_ERROR", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) parseIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return uuid.Nil, uuid.Nil, false
	}
	findingID, err := uuid.Parse(chi.URLParam(r, "findingId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid finding id")
		return uuid.Nil, uuid.Nil, false
	}
	return caseID, findingID, true
}

func (h *Handler) writeTransitionResponse(w http.ResponseWriter, finding FindingResponse, err error) {
	if err != nil {
		switch err {
		case ErrNotFound:
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "finding not found")
		case ErrForbidden:
			response.Err(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		case ErrInvalidStatus:
			response.Err(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
		case ErrEvidenceRequired:
			response.Err(w, http.StatusUnprocessableEntity, "EVIDENCE_REQUIRED", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "TRANSITION_ERROR", err.Error())
		}
		return
	}
	response.JSON(w, http.StatusOK, finding)
}
