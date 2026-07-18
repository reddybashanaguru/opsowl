package subjects

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

// ── Subject CRUD ──────────────────────────────────────────────────────────────

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilters{
		Search:     q.Get("q"),
		EntityType: q.Get("entity_type"),
		RiskLevel:  q.Get("risk_level"),
	}
	if v := q.Get("is_pep"); v != "" {
		b, _ := strconv.ParseBool(v)
		f.IsPEP = &b
	}
	if v := q.Get("is_sanctioned"); v != "" {
		b, _ := strconv.ParseBool(v)
		f.IsSanctioned = &b
	}
	if v := q.Get("watchlisted"); v != "" {
		b, _ := strconv.ParseBool(v)
		f.Watchlisted = &b
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	f.Limit = limit
	f.Offset = offset

	subjects, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list subjects")
		return
	}
	hasMore := int64(f.Offset+len(subjects)) < total
	response.JSONWithMeta(w, http.StatusOK, subjects, response.Meta{HasMore: hasMore, Total: &total})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	var req CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.CanonicalName == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "canonical_name is required")
		return
	}
	sub, err := h.svc.Create(r.Context(), claims, req)
	if err != nil {
		if errors.Is(err, ErrDuplicateID) {
			response.Err(w, http.StatusConflict, "DUPLICATE_SUBJECT", err.Error())
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create subject")
		return
	}
	response.JSON(w, http.StatusCreated, sub)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	sub, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get subject")
		return
	}
	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	var req UpdateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	sub, err := h.svc.Update(r.Context(), claims, id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update subject")
		return
	}
	response.JSON(w, http.StatusOK, sub)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	if err := h.svc.Delete(r.Context(), claims, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete subject")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Subject 360 aggregated data ───────────────────────────────────────────────

func (h *Handler) GetCases(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	cases, err := h.svc.CasesForSubject(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get cases")
		return
	}
	if cases == nil {
		cases = []SubjectCaseRef{}
	}
	response.JSON(w, http.StatusOK, map[string]any{"cases": cases})
}

func (h *Handler) GetFilings(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	filings, err := h.svc.FilingsForSubject(r.Context(), id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get filings")
		return
	}
	response.JSON(w, http.StatusOK, filings)
}

func (h *Handler) GetRiskHistory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	history, err := h.svc.RiskHistory(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get risk history")
		return
	}
	response.JSON(w, http.StatusOK, history)
}

func (h *Handler) SetWatchlist(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	var body struct {
		Watchlisted bool `json:"watchlisted"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	sub, err := h.svc.SetWatchlist(r.Context(), claims, id, body.Watchlisted)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update watchlist")
		return
	}
	response.JSON(w, http.StatusOK, sub)
}

// ── Case ↔ Subject linking ────────────────────────────────────────────────────

func (h *Handler) ListForCase(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	subjects, err := h.svc.ListForCase(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list subjects")
		return
	}
	response.JSON(w, http.StatusOK, subjects)
}

func (h *Handler) LinkToCase(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req struct {
		SubjectID *uuid.UUID           `json:"subject_id"`
		Role      string               `json:"role"`
		Subject   *CreateSubjectRequest `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	var subjectID uuid.UUID
	if req.SubjectID != nil {
		subjectID = *req.SubjectID
	} else if req.Subject != nil {
		// Create-and-link: resolve or create subject then link
		sub, _, err := h.svc.ResolveOrCreate(r.Context(), claims, *req.Subject)
		if err != nil {
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resolve subject")
			return
		}
		subjectID = sub.ID
	} else {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "subject_id or subject object is required")
		return
	}

	role := req.Role
	if role == "" {
		role = "primary"
	}
	if err := h.svc.LinkToCase(r.Context(), claims, caseID, subjectID, role); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
		case errors.Is(err, ErrAlreadyLinked):
			response.Err(w, http.StatusConflict, "ALREADY_LINKED", err.Error())
		case errors.Is(err, ErrCaseClosed):
			response.Err(w, http.StatusConflict, "CASE_CLOSED", err.Error())
		default:
			response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to link subject")
		}
		return
	}
	// Return updated subject list for the case
	subjects, _ := h.svc.ListForCase(r.Context(), caseID)
	response.JSON(w, http.StatusOK, subjects)
}

func (h *Handler) UnlinkFromCase(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "caseId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}
	subjectID, err := uuid.Parse(chi.URLParam(r, "subjectId"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	if err := h.svc.UnlinkFromCase(r.Context(), caseID, subjectID); err != nil {
		if errors.Is(err, ErrNotLinked) {
			response.Err(w, http.StatusNotFound, "NOT_LINKED", "subject is not linked to this case")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to unlink subject")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Fuzzy Entity Resolution ───────────────────────────────────────────────────

// GET /api/v1/subjects/:id/similar?threshold=0.3&limit=10
func (h *Handler) GetSimilar(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	var threshold float64
	if v := r.URL.Query().Get("threshold"); v != "" {
		threshold, _ = strconv.ParseFloat(v, 64)
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	results, err := h.svc.FindSimilar(r.Context(), id, threshold, limit)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Err(w, http.StatusNotFound, "NOT_FOUND", "subject not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, results)
}

// POST /api/v1/subjects/:id/confirm-match
func (h *Handler) ConfirmMatch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid subject id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	var req ConfirmMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if err := h.svc.ConfirmMatch(r.Context(), claims, id, req); err != nil {
		response.Err(w, http.StatusBadRequest, "CONFIRM_MATCH_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}
