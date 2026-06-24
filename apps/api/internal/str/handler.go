package str

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/cases"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
	"github.com/reddybashanaguru/opsowl/internal/timeline"
)

type Handler struct {
	repo    *Repository
	emitter timeline.Emitter
}

func NewHandler(repo *Repository, emitter timeline.Emitter) *Handler {
	return &Handler{repo: repo, emitter: emitter}
}

func (h *Handler) emitSTR(caseID uuid.UUID, actorID uuid.UUID, evtType db.TimelineEventType, f *filing) {
	payload := map[string]any{
		"str_id":      f.ID.String(),
		"report_type": f.ReportType,
		"status":      f.Status,
	}
	_ = h.emitter.Emit(context.Background(), timeline.Event{
		CaseID:     caseID,
		EventType:  evtType,
		Source:     db.TimelineEventSourceUser,
		ActorID:    &actorID,
		OccurredAt: time.Now(),
		Payload:    payload,
	})
}

// POST /api/v1/cases/{id}/str
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.SuspicionBasis == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "suspicion_basis is required")
		return
	}

	f, err := h.repo.Create(r.Context(), caseID, cases.DefaultOrgID, req)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create STR draft")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	h.emitSTR(caseID, claims.UserID, db.TimelineEventTypeStrDraftCreated, f)
	response.JSON(w, http.StatusCreated, toResponse(f))
}

// GET /api/v1/cases/{id}/str
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	filings, err := h.repo.ListByCaseID(r.Context(), caseID, cases.DefaultOrgID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list STR filings")
		return
	}

	out := make([]*FilingResponse, 0, len(filings))
	for i := range filings {
		out = append(out, toResponse(&filings[i]))
	}
	response.JSON(w, http.StatusOK, map[string]any{"filings": out})
}

// GET /api/v1/cases/{id}/str/{str_id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), strID, cases.DefaultOrgID)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "STR filing not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch STR filing")
		return
	}
	response.JSON(w, http.StatusOK, toResponse(f))
}

// POST /api/v1/cases/{id}/str/{str_id}/submit
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	f, err := h.repo.Submit(r.Context(), strID, cases.DefaultOrgID, claims.UserID)
	if errors.Is(err, ErrWrongStatus) {
		response.Err(w, http.StatusConflict, "WRONG_STATUS", "filing must be in draft status to submit")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to submit STR filing")
		return
	}
	h.emitSTR(f.CaseID, claims.UserID, db.TimelineEventTypeStrSubmitted, f)
	response.JSON(w, http.StatusOK, toResponse(f))
}

// POST /api/v1/cases/{id}/str/{str_id}/approve
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())

	f, err := h.repo.Approve(r.Context(), strID, cases.DefaultOrgID, claims.UserID)
	if errors.Is(err, ErrMakerChecker) {
		response.Err(w, http.StatusForbidden, "MAKER_CHECKER_VIOLATION",
			"AML/CFT regulations: the approver must be a different person from the submitter")
		return
	}
	if errors.Is(err, ErrWrongStatus) {
		response.Err(w, http.StatusConflict, "WRONG_STATUS", "filing must be in submitted status to approve")
		return
	}
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "STR filing not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to approve STR filing")
		return
	}
	h.emitSTR(f.CaseID, claims.UserID, db.TimelineEventTypeStrApproved, f)
	response.JSON(w, http.StatusOK, toResponse(f))
}

// POST /api/v1/cases/{id}/str/{str_id}/reject
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}
	var req RejectRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	f, err := h.repo.Reject(r.Context(), strID, cases.DefaultOrgID, req.Reason)
	if errors.Is(err, ErrWrongStatus) {
		response.Err(w, http.StatusConflict, "WRONG_STATUS", "filing must be in submitted status to reject")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to reject STR filing")
		return
	}
	rejectClaims := auth.ClaimsFromCtx(r.Context())
	h.emitSTR(f.CaseID, rejectClaims.UserID, db.TimelineEventTypeStrRejected, f)
	response.JSON(w, http.StatusOK, toResponse(f))
}

// POST /api/v1/cases/{id}/str/{str_id}/file
func (h *Handler) MarkFiled(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}
	var req MarkFiledRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	f, err := h.repo.MarkFiled(r.Context(), strID, cases.DefaultOrgID, req.FIUReference)
	if errors.Is(err, ErrWrongStatus) {
		response.Err(w, http.StatusConflict, "WRONG_STATUS", "filing must be approved before marking as filed")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to mark STR as filed")
		return
	}
	fileClaims := auth.ClaimsFromCtx(r.Context())
	h.emitSTR(f.CaseID, fileClaims.UserID, db.TimelineEventTypeStrFiled, f)
	response.JSON(w, http.StatusOK, toResponse(f))
}

// GET /api/v1/cases/{id}/str/{str_id}/validate/goaml
// Returns a JSON list of validation errors — empty array means ready to export.
func (h *Handler) ValidateGoAML(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), strID, cases.DefaultOrgID)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "STR filing not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch STR filing")
		return
	}

	errs := ValidateGoAMLReadiness(toResponse(f))
	response.JSON(w, http.StatusOK, map[string]any{
		"ready":  len(errs) == 0,
		"errors": errs,
	})
}

// PATCH /api/v1/cases/{id}/str/{str_id}/eu-indicators
// Saves manually selected EU FATF typology codes (compliance officer confirms/overrides).
func (h *Handler) SaveEUIndicators(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}
	var body struct {
		Indicators        []string `json:"indicators"`
		SubmissionCountry string   `json:"submission_country"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if err := h.repo.SaveEUIndicators(r.Context(), strID, cases.DefaultOrgID, body.Indicators, body.SubmissionCountry); err != nil {
		response.Err(w, http.StatusInternalServerError, "SAVE_FAILED", err.Error())
		return
	}
	f, _ := h.repo.GetByID(r.Context(), strID, cases.DefaultOrgID)
	response.JSON(w, http.StatusOK, toResponse(f))
}

// GET /api/v1/cases/{id}/str/{str_id}/export/goaml
func (h *Handler) ExportGoAML(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), strID, cases.DefaultOrgID)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "STR filing not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch STR filing")
		return
	}

	// EU validation: Hellenic FIU (and all EU FIUs) require the entity FIU registration number
	if strings.ToLower(os.Getenv("DEPLOYMENT_JURISDICTION")) == "eu" {
		var entity goAMLEntityJSON
		_ = json.Unmarshal(f.FilingEntity, &entity)
		if entity.RentityID == "" {
			regulator := os.Getenv("STR_REGULATOR_LABEL")
			if regulator == "" {
				regulator = "FIU"
			}
			response.Err(w, http.StatusUnprocessableEntity, "EU_ENTITY_REQUIRED",
				regulator+" registration number (filing_entity.rentity_id) is required for EU submission")
			return
		}
	}

	xmlData, validationErrs := BuildGoAMLXML(toResponse(f))
	if len(validationErrs) > 0 {
		response.JSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "GOAML_VALIDATION_FAILED",
			"errors": validationErrs,
		})
		return
	}

	filename := fmt.Sprintf("goAML-STR-%s.xml", strID.String()[:8])
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlData)
}

// GET /api/v1/cases/{id}/str/{str_id}/export
func (h *Handler) ExportFINnet(w http.ResponseWriter, r *http.Request) {
	strID, err := uuid.Parse(chi.URLParam(r, "str_id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid str filing id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), strID, cases.DefaultOrgID)
	if errors.Is(err, ErrNotFound) {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "STR filing not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch STR filing")
		return
	}

	xmlData, err := BuildFINnetXML(toResponse(f))
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "XML_ERROR", "failed to generate FINnet XML")
		return
	}

	filename := fmt.Sprintf("STR-%s.xml", strID.String()[:8])
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(xmlData)
}

// GET /api/v1/str/deadlines?deadline_days=30&window_days=14
// Returns STR filings with approaching or overdue deadlines across all cases.
func (h *Handler) Deadlines(w http.ResponseWriter, r *http.Request) {
	ddStr := r.URL.Query().Get("deadline_days")
	wdStr  := r.URL.Query().Get("window_days")

	dd := 30
	wd := 14
	if v, err := strconv.Atoi(ddStr); err == nil && v > 0 {
		dd = v
	}
	if v, err := strconv.Atoi(wdStr); err == nil && v > 0 {
		wd = v
	}

	items, err := h.repo.Deadlines(r.Context(), cases.DefaultOrgID, dd, wd)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to fetch STR deadlines")
		return
	}
	if items == nil {
		items = []DeadlineItem{}
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"deadline_days": dd,
		"window_days":   wd,
		"count":         len(items),
		"items":         items,
	})
}
