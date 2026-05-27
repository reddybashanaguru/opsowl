package analytics

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// GET /api/v1/analytics/workload/export
func (h *Handler) ExportWorkload(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.GetAnalystWorkload(r.Context())
	if err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	filename := fmt.Sprintf("workload-%s.csv", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"full_name", "email", "role", "active_cases", "open_tasks", "overdue_tasks", "avg_case_age_hours"})
	for _, row := range rows {
		_ = cw.Write([]string{
			row.FullName,
			row.Email,
			string(row.Role),
			strconv.FormatInt(row.ActiveCases, 10),
			strconv.FormatInt(row.OpenTasks, 10),
			strconv.FormatInt(row.OverdueTasks, 10),
			row.AvgCaseAgeHours,
		})
	}
	cw.Flush()
}

// riskRowDTO is the JSON-safe projection of ListCasesWithRiskScoreRow.
// sql.NullString / sql.NullTime must not be marshalled directly — they produce
// {"String":"...","Valid":true} objects which React cannot render as children.
type riskRowDTO struct {
	ID          string  `json:"id"`
	CaseNumber  string  `json:"case_number"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	SlaBreached bool    `json:"sla_breached"`
	RiskScore   *string `json:"risk_score"`
	RiskLevel   *string `json:"risk_level"`
	AnalysedAt  *string `json:"analysed_at"`
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

	dtos := make([]riskRowDTO, len(rows))
	for i, row := range rows {
		dto := riskRowDTO{
			ID:          row.ID.String(),
			CaseNumber:  row.CaseNumber,
			Title:       row.Title,
			Status:      string(row.Status),
			Priority:    string(row.Priority),
			SlaBreached: row.SlaBreached,
		}
		if row.RiskScore.Valid {
			dto.RiskScore = &row.RiskScore.String
		}
		if row.RiskLevel.Valid {
			dto.RiskLevel = &row.RiskLevel.String
		}
		if row.AnalysedAt.Valid {
			s := row.AnalysedAt.Time.UTC().Format(time.RFC3339)
			dto.AnalysedAt = &s
		}
		dtos[i] = dto
	}
	response.JSON(w, http.StatusOK, dtos)
}

// GET /api/v1/analytics/risk/export
func (h *Handler) ExportRisk(w http.ResponseWriter, r *http.Request) {
	limit := int32(100)
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.ParseInt(v, 10, 32); err == nil && l > 0 && l <= 1000 {
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

	filename := fmt.Sprintf("risk-leaderboard-%s.csv", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"case_number", "title", "status", "priority", "sla_breached", "risk_score", "risk_level", "analysed_at"})
	for _, row := range rows {
		riskScore := ""
		if row.RiskScore.Valid {
			riskScore = row.RiskScore.String
		}
		riskLevel := ""
		if row.RiskLevel.Valid {
			riskLevel = row.RiskLevel.String
		}
		analysedAt := ""
		if row.AnalysedAt.Valid {
			analysedAt = row.AnalysedAt.Time.UTC().Format(time.RFC3339)
		}
		slaBreached := "false"
		if row.SlaBreached {
			slaBreached = "true"
		}
		_ = cw.Write([]string{
			row.CaseNumber,
			row.Title,
			string(row.Status),
			string(row.Priority),
			slaBreached,
			riskScore,
			riskLevel,
			analysedAt,
		})
	}
	cw.Flush()
}
