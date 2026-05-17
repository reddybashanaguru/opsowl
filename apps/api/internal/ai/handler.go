package ai

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

func (h *Handler) Ask(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.Ask(r.Context(), claims, caseID, req.Question)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Summarize(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.Summarize(r.Context(), claims, caseID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) GenerateNote(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.GenerateNote(r.Context(), claims, caseID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ExtractActions(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req ExtractActionsRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional
	if req.SourceType == "" {
		req.SourceType = "documents"
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.ExtractActions(r.Context(), claims, caseID, req.SourceType)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	q := r.URL.Query()
	status := q.Get("status")
	limit := int32(20)
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = int32(l)
	}

	var cursorID *uuid.UUID
	if c := q.Get("cursor"); c != "" {
		if parsed, err := uuid.Parse(c); err == nil {
			cursorID = &parsed
		}
	}

	logs, meta, err := h.svc.ListByCase(r.Context(), caseID, ListLogsRequest{
		Status:   status,
		CursorID: cursorID,
		Limit:    limit,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, logs, map[string]interface{}{
		"total_count": meta.TotalCount,
		"has_more":    meta.HasMore,
		"next_cursor": meta.NextCursor,
	})
}

func (h *Handler) GetLog(w http.ResponseWriter, r *http.Request) {
	logID, err := uuid.Parse(chi.URLParam(r, "logId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid log id")
		return
	}

	result, err := h.svc.GetLog(r.Context(), logID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	logID, err := uuid.Parse(chi.URLParam(r, "logId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid log id")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.Approve(r.Context(), claims, logID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	logID, err := uuid.Parse(chi.URLParam(r, "logId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid log id")
		return
	}

	var req RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.Reject(r.Context(), claims, logID, req.Reason)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	logID, err := uuid.Parse(chi.URLParam(r, "logId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid log id")
		return
	}

	var req EditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	result, err := h.svc.Edit(r.Context(), claims, logID, req.EditedResponse)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	logs, err := h.svc.ListPending(r.Context(), 50)
	if err != nil {
		h.handleError(w, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, logs, map[string]interface{}{"count": len(logs)})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "AI log not found")
	case errors.Is(err, ErrForbidden):
		response.Err(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
	case errors.Is(err, ErrBadRequest):
		response.ErrDetail(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
	default:
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
