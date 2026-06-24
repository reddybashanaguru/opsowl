package ubo

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reddybashanaguru/opsowl/internal/auth"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /api/v1/cases/{caseId}/ubo
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	records, err := h.svc.repo.ListByCaseID(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if records == nil {
		records = []*UBORecord{}
	}
	response.JSON(w, http.StatusOK, records)
}

// POST /api/v1/cases/{caseId}/ubo
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	claims := auth.ClaimsFromCtx(r.Context())

	var req UBORecord
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}
	if req.EntityName == "" || req.OwnerName == "" {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "entity_name and owner_name are required")
		return
	}
	if req.OwnershipPct <= 0 || req.OwnershipPct > 100 {
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "ownership_pct must be between 0 and 100")
		return
	}

	req.CaseID = caseID
	rec, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	_ = claims
	response.JSON(w, http.StatusCreated, rec)
}

type updateStatusRequest struct {
	Status          string `json:"status"`
	DiscrepancyNote string `json:"discrepancy_note"`
}

// PATCH /api/v1/cases/{caseId}/ubo/{id}
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims := auth.ClaimsFromCtx(r.Context())

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	status := VerificationStatus(req.Status)
	switch status {
	case StatusVerified, StatusDiscrepant, StatusNotApplicable:
		// valid
	default:
		response.Err(w, http.StatusBadRequest, "VALIDATION_ERROR", "status must be verified, discrepant, or n/a")
		return
	}

	err := h.svc.repo.UpdateStatus(r.Context(), UpdateStatusParams{
		ID:              id,
		Status:          status,
		VerifiedBy:      claims.UserID.String(),
		DiscrepancyNote: req.DiscrepancyNote,
	})
	if err == sql.ErrNoRows {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "ubo record not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/cases/{caseId}/ubo/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.svc.repo.Delete(r.Context(), id)
	if err == sql.ErrNoRows {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "ubo record not found")
		return
	}
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
