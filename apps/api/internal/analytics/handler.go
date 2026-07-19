package analytics

import (
	"context"
	"database/sql"
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
	rawDB   *sql.DB
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{queries: queries}
}

func NewHandlerWithDB(queries *db.Queries, rawDB *sql.DB) *Handler {
	return &Handler{queries: queries, rawDB: rawDB}
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

// MLROSummaryResponse is the payload for GET /analytics/mlro-summary.
type MLROSummaryResponse struct {
	CasesOpenedMTD    int64            `json:"cases_opened_mtd"`
	CasesClosedMTD    int64            `json:"cases_closed_mtd"`
	STRsFiledMTD      int64            `json:"strs_filed_mtd"`
	STRsPendingReview int64            `json:"strs_pending_review"`
	SLABreachedOpen   int64            `json:"sla_breached_open"`
	AvgCaseAgeDays    int64            `json:"avg_case_age_days"`
	DispositionCounts map[string]int64 `json:"disposition_counts"`
	Investigators     []InvestigatorKPI `json:"investigators"`
}

type InvestigatorKPI struct {
	UserID       string `json:"user_id"`
	FullName     string `json:"full_name"`
	ActiveCases  int64  `json:"active_cases"`
	ClosedMTD    int64  `json:"closed_mtd"`
	STRsMTD      int64  `json:"strs_mtd"`
	BreachedOpen int64  `json:"breached_open"`
}

// GET /api/v1/analytics/mlro-summary
func (h *Handler) MLROSummary(w http.ResponseWriter, r *http.Request) {
	if h.rawDB == nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", "db not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var out MLROSummaryResponse
	out.DispositionCounts = make(map[string]int64)
	out.Investigators = []InvestigatorKPI{}

	statsQ := `
		SELECT
			(SELECT COUNT(*) FROM cases WHERE created_at >= date_trunc('month', NOW()) AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM cases WHERE status = 'closed' AND updated_at >= date_trunc('month', NOW()) AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM str_filings WHERE status IN ('approved','filed') AND updated_at >= date_trunc('month', NOW())),
			(SELECT COUNT(*) FROM str_filings WHERE status IN ('draft','submitted')),
			(SELECT COUNT(*) FROM cases WHERE sla_breached = true AND status NOT IN ('closed','draft') AND deleted_at IS NULL),
			(SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (NOW() - created_at))/86400), 0)::bigint FROM cases WHERE status NOT IN ('closed','draft') AND deleted_at IS NULL)`
	row := h.rawDB.QueryRowContext(ctx, statsQ)
	if err := row.Scan(
		&out.CasesOpenedMTD, &out.CasesClosedMTD,
		&out.STRsFiledMTD, &out.STRsPendingReview,
		&out.SLABreachedOpen, &out.AvgCaseAgeDays,
	); err != nil {
		response.Err(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	dispQ := `SELECT cc.disposition, COUNT(*) FROM case_closures cc WHERE cc.status = 'approved' GROUP BY cc.disposition`
	if drows, err := h.rawDB.QueryContext(ctx, dispQ); err == nil {
		defer drows.Close()
		for drows.Next() {
			var disp string
			var cnt int64
			if err := drows.Scan(&disp, &cnt); err == nil {
				out.DispositionCounts[disp] = cnt
			}
		}
	}

	kpiQ := `
		SELECT u.id::text, u.full_name,
			COALESCE(SUM(CASE WHEN c.status NOT IN ('closed','draft') AND c.deleted_at IS NULL THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN c.status = 'closed' AND c.updated_at >= date_trunc('month', NOW()) THEN 1 ELSE 0 END), 0),
			COALESCE(COUNT(DISTINCT CASE WHEN sr.status IN ('approved','filed') AND sr.updated_at >= date_trunc('month', NOW()) THEN sr.id END), 0),
			COALESCE(SUM(CASE WHEN c.sla_breached = true AND c.status NOT IN ('closed','draft') AND c.deleted_at IS NULL THEN 1 ELSE 0 END), 0)
		FROM users u
		LEFT JOIN cases c ON c.assigned_to = u.id
		LEFT JOIN str_filings sr ON sr.submitted_by = u.id
		WHERE u.role IN ('analyst','manager') AND u.is_active = true
		GROUP BY u.id, u.full_name
		ORDER BY 3 DESC, u.full_name`
	if krows, err := h.rawDB.QueryContext(ctx, kpiQ); err == nil {
		defer krows.Close()
		for krows.Next() {
			var kpi InvestigatorKPI
			if err := krows.Scan(&kpi.UserID, &kpi.FullName, &kpi.ActiveCases, &kpi.ClosedMTD, &kpi.STRsMTD, &kpi.BreachedOpen); err == nil {
				out.Investigators = append(out.Investigators, kpi)
			}
		}
	}

	response.JSON(w, http.StatusOK, out)
}

// GET /api/v1/analytics/mlro-summary/export — CSV for monthly regulatory reporting
func (h *Handler) ExportMLROSummary(w http.ResponseWriter, r *http.Request) {
	var data MLROSummaryResponse
	data.DispositionCounts = make(map[string]int64)
	data.Investigators = []InvestigatorKPI{}

	// reuse the same queries as MLROSummary
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if h.rawDB != nil {
		statsQ := `
			SELECT
				(SELECT COUNT(*) FROM cases WHERE created_at >= date_trunc('month', NOW()) AND deleted_at IS NULL),
				(SELECT COUNT(*) FROM cases WHERE status = 'closed' AND updated_at >= date_trunc('month', NOW()) AND deleted_at IS NULL),
				(SELECT COUNT(*) FROM str_filings WHERE status IN ('approved','filed') AND updated_at >= date_trunc('month', NOW())),
				(SELECT COUNT(*) FROM str_filings WHERE status IN ('draft','submitted')),
				(SELECT COUNT(*) FROM cases WHERE sla_breached = true AND status NOT IN ('closed','draft') AND deleted_at IS NULL),
				(SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (NOW() - created_at))/86400), 0)::bigint FROM cases WHERE status NOT IN ('closed','draft') AND deleted_at IS NULL)`
		_ = h.rawDB.QueryRowContext(ctx, statsQ).Scan(
			&data.CasesOpenedMTD, &data.CasesClosedMTD,
			&data.STRsFiledMTD, &data.STRsPendingReview,
			&data.SLABreachedOpen, &data.AvgCaseAgeDays,
		)
		if krows, err := h.rawDB.QueryContext(ctx, `
			SELECT u.id::text, u.full_name,
				COALESCE(SUM(CASE WHEN c.status NOT IN ('closed','draft') AND c.deleted_at IS NULL THEN 1 ELSE 0 END), 0),
				COALESCE(SUM(CASE WHEN c.status = 'closed' AND c.updated_at >= date_trunc('month', NOW()) THEN 1 ELSE 0 END), 0),
				COALESCE(COUNT(DISTINCT CASE WHEN sr.status IN ('approved','filed') AND sr.updated_at >= date_trunc('month', NOW()) THEN sr.id END), 0),
				COALESCE(SUM(CASE WHEN c.sla_breached = true AND c.status NOT IN ('closed','draft') AND c.deleted_at IS NULL THEN 1 ELSE 0 END), 0)
			FROM users u
			LEFT JOIN cases c ON c.assigned_to = u.id
			LEFT JOIN str_filings sr ON sr.submitted_by = u.id
			WHERE u.role IN ('analyst','manager') AND u.is_active = true
			GROUP BY u.id, u.full_name ORDER BY 3 DESC, u.full_name`); err == nil {
			defer krows.Close()
			for krows.Next() {
				var kpi InvestigatorKPI
				if err := krows.Scan(&kpi.UserID, &kpi.FullName, &kpi.ActiveCases, &kpi.ClosedMTD, &kpi.STRsMTD, &kpi.BreachedOpen); err == nil {
					data.Investigators = append(data.Investigators, kpi)
				}
			}
		}
	}

	month := time.Now().UTC().Format("2006-01")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="mlro-report-`+month+`.csv"`)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"metric", "value"})
	_ = cw.Write([]string{"period", month})
	_ = cw.Write([]string{"cases_opened_mtd", strconv.FormatInt(data.CasesOpenedMTD, 10)})
	_ = cw.Write([]string{"cases_closed_mtd", strconv.FormatInt(data.CasesClosedMTD, 10)})
	_ = cw.Write([]string{"strs_filed_mtd", strconv.FormatInt(data.STRsFiledMTD, 10)})
	_ = cw.Write([]string{"strs_pending_review", strconv.FormatInt(data.STRsPendingReview, 10)})
	_ = cw.Write([]string{"sla_breached_open", strconv.FormatInt(data.SLABreachedOpen, 10)})
	_ = cw.Write([]string{"avg_case_age_days", strconv.FormatInt(data.AvgCaseAgeDays, 10)})
	_ = cw.Write([]string{})
	_ = cw.Write([]string{"investigator", "active_cases", "closed_mtd", "strs_mtd", "sla_breached"})
	for _, inv := range data.Investigators {
		_ = cw.Write([]string{
			inv.FullName,
			strconv.FormatInt(inv.ActiveCases, 10),
			strconv.FormatInt(inv.ClosedMTD, 10),
			strconv.FormatInt(inv.STRsMTD, 10),
			strconv.FormatInt(inv.BreachedOpen, 10),
		})
	}
	cw.Flush()
}
