package ai

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

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

func (h *Handler) RunAgent(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req AgentRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Goal == "" {
		response.Err(w, http.StatusBadRequest, "BAD_REQUEST", "goal is required")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	// StreamAgent writes headers only after the Python service confirms 200.
	// Any error returned here means headers have NOT been written yet — safe to write JSON error.
	if err := h.svc.StreamAgent(r.Context(), claims, caseID, req.Goal, w); err != nil {
		h.svc.logger.Warn("agent stream error", zap.String("case_id", caseID.String()), zap.Error(err))
		h.handleError(w, err)
	}
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

// ExportCSV streams an AI Act audit trail CSV for the given date range.
// EU AI Act Art. 13 (transparency) and Art. 14 (human oversight) require
// documented audit trails for all high-risk AI system decisions.
// GET /api/v1/ai/logs/export?from=2024-01-01&to=2024-12-31
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromStr := q.Get("from")
	toStr := q.Get("to")

	from := time.Now().AddDate(0, -1, 0) // default: last 30 days
	to := time.Now()

	if fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t.Add(24 * time.Hour) // inclusive of end date
		}
	}

	logs, err := h.svc.ListForExport(r.Context(), from, to)
	if err != nil {
		h.handleError(w, err)
		return
	}

	filename := fmt.Sprintf("ai-act-audit-%s-to-%s.csv",
		from.Format("2006-01-02"), to.Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	enc := csv.NewWriter(w)
	_ = enc.Write([]string{
		"log_id", "case_id", "case_number", "action_type",
		"model_used", "provider", "confidence_score",
		"human_review_status", "reviewed_by", "reviewed_at",
		"created_at",
	})
	for _, l := range logs {
		reviewedBy := ""
		if l.ReviewedBy != nil {
			reviewedBy = l.ReviewedBy.String()
		}
		reviewedAt := ""
		if l.ReviewedAt != nil {
			reviewedAt = l.ReviewedAt.Format(time.RFC3339)
		}
		conf := ""
		if l.ConfidenceScore != nil {
			conf = fmt.Sprintf("%.4f", *l.ConfidenceScore)
		}
		_ = enc.Write([]string{
			l.ID.String(),
			l.CaseID.String(),
			l.CaseNumber,
			l.ActionType,
			l.ModelUsed,
			l.Provider,
			conf,
			l.HumanReviewStatus,
			reviewedBy,
			reviewedAt,
			l.CreatedAt.Format(time.RFC3339),
		})
	}
	enc.Flush()
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
