package analytics

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

// GET /api/v1/analytics/workload
func (h *Handler) Workload(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.GetAnalystWorkload(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rows)
}

// GET /api/v1/analytics/risk  — top N cases by AI risk score
func (h *Handler) RiskLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := int32(20)
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.ParseInt(v, 10, 32); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	rows, err := h.queries.ListCasesWithRiskScore(r.Context(), db.ListCasesWithRiskScoreParams{
		OrganizationID: defaultOrgID,
		RowLimit:       limit,
	})
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rows)
}
