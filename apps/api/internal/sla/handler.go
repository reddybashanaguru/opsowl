package sla

import (
	"encoding/json"
	"net/http"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

type upsertSLARequest struct {
	Priority     string `json:"priority"`
	WarningHours int32  `json:"warning_hours"`
	BreachHours  int32  `json:"breach_hours"`
	WarnPct      int32  `json:"warn_pct"`
}

// GET /api/v1/sla/policies
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	policies, err := h.queries.ListSLAPolicies(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, policies)
}

// PUT /api/v1/sla/policies
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	var req upsertSLARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Priority == "" || req.BreachHours <= 0 {
		response.Err(w, http.StatusBadRequest, "MISSING_FIELDS", "priority and breach_hours are required")
		return
	}

	policy, err := h.queries.UpsertSLAPolicy(r.Context(), db.UpsertSLAPolicyParams{
		Priority:     req.Priority,
		WarningHours: req.WarningHours,
		BreachHours:  req.BreachHours,
		WarnPct:      req.WarnPct,
	})
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, policy)
}
