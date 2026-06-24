package gdpr

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc  *Service
	repo *Repository
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, repo: NewRepository(svc.db)}
}

// List returns all DSAR requests ordered by deadline (soonest first).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	if items == nil {
		items = []*DSAR{}
	}
	response.JSON(w, http.StatusOK, items)
}

// Create registers a new DSAR and computes the 30-day GDPR response deadline.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubjectName  string `json:"subject_name"`
		SubjectEmail string `json:"subject_email"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if body.SubjectName == "" {
		response.Err(w, http.StatusBadRequest, "MISSING_SUBJECT_NAME", "subject_name is required")
		return
	}
	dsar, err := h.svc.CreateDSAR(r.Context(), body.SubjectName, body.SubjectEmail, body.Notes)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, dsar)
}

// UpdateStatus moves a DSAR between lifecycle states (in_progress, deferred, rejected).
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status     string `json:"status"`
		LegalBasis string `json:"legal_basis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	claims := auth.ClaimsFromCtx(r.Context())
	userID := ""
	if claims != nil {
		userID = claims.UserID.String()
	}
	if err := h.repo.UpdateStatus(r.Context(), id, body.Status, body.LegalBasis, userID); err != nil {
		response.Err(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	dsar, err := h.repo.Get(r.Context(), id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, dsar)
}

// Process triggers pseudonymisation/erasure for the data subject named in the DSAR.
// This is irreversible — confirmation modal on the UI is mandatory before calling this.
func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	dsar, err := h.repo.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Err(w, http.StatusNotFound, "DSAR_NOT_FOUND", "DSAR not found")
			return
		}
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	if dsar.Status == DSARCompleted || dsar.Status == DSARRejected {
		response.Err(w, http.StatusConflict, "ALREADY_PROCESSED", "DSAR has already been processed")
		return
	}

	claims := auth.ClaimsFromCtx(r.Context())
	userID := ""
	if claims != nil {
		userID = claims.UserID.String()
	}

	if err := h.svc.ProcessDSAR(r.Context(), id, userID); err != nil {
		response.Err(w, http.StatusInternalServerError, "PROCESS_FAILED", err.Error())
		return
	}

	updated, err := h.repo.Get(r.Context(), id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}

	records, _ := h.repo.ListAnonymisationRecords(r.Context(), id)
	response.JSON(w, http.StatusOK, map[string]any{
		"dsar":    updated,
		"records": records,
	})
}

// GetAnonymisationRecords returns the audit trail for a processed DSAR.
func (h *Handler) GetAnonymisationRecords(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	records, err := h.repo.ListAnonymisationRecords(r.Context(), id)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	if records == nil {
		records = []AnonymisationRecord{}
	}
	response.JSON(w, http.StatusOK, records)
}
