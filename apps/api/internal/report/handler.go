package report

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"go.uber.org/zap"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
	"github.com/reddybashanaguru/opsowl/internal/response"
)

type Handler struct {
	queries      *db.Queries
	aiServiceURL string
	httpClient   *http.Client
	logger       *zap.Logger
}

func NewHandler(queries *db.Queries, aiServiceURL string, logger *zap.Logger) *Handler {
	return &Handler{
		queries:      queries,
		aiServiceURL: aiServiceURL,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		logger:       logger,
	}
}

// GET /api/v1/cases/:id/report
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	caseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Err(w, http.StatusBadRequest, "INVALID_ID", "invalid case id")
		return
	}

	// Gather all data in parallel-ish (sequential for simplicity, fast enough)
	caseRow, err := h.queries.GetCaseByID(r.Context(), caseID)
	if err != nil {
		response.Err(w, http.StatusNotFound, "NOT_FOUND", "case not found")
		return
	}

	tasks, _ := h.queries.ListTasksByCase(r.Context(), db.ListTasksByCaseParams{
		CaseID:   caseID,
		RowLimit: 100,
	})
	timeline, _ := h.queries.ListTimelineEvents(r.Context(), db.ListTimelineEventsParams{
		CaseID:   caseID,
		RowLimit: 100,
	})
	documents, _ := h.queries.ListDocumentsByCase(r.Context(), db.ListDocumentsByCaseParams{
		CaseID:   caseID,
		RowLimit: 100,
	})
	notes, _ := h.queries.ListNotesByCase(r.Context(), db.ListNotesByCaseParams{
		CaseID:  caseID,
		Column2: nil,
		Column3: uuid.Nil,
		Limit:   50,
	})
	comms, _ := h.queries.ListCommLogs(r.Context(), db.ListCommLogsParams{
		CaseID:   caseID,
		RowLimit: 50,
	})

	// AI executive summary (graceful fallback)
	summary := h.fetchAISummary(r.Context(), caseRow, len(tasks), len(documents))

	// AI insights (latest)
	insights, _ := h.queries.GetLatestCaseAIInsights(r.Context(), caseID)

	pdf := buildPDF(caseRow, summary, tasks, timeline, documents, notes, comms, insights)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		h.logger.Error("pdf generation failed", zap.Error(err))
		response.Err(w, http.StatusInternalServerError, "PDF_ERROR", "failed to generate report")
		return
	}

	filename := fmt.Sprintf("%s-report.pdf", strings.ReplaceAll(caseRow.CaseNumber, "/", "-"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// ── AI summary ────────────────────────────────────────────────────────────────

type aiSummaryRequest struct {
	CaseNumber   string `json:"case_number"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Priority     string `json:"priority"`
	Description  string `json:"description"`
	TaskCount    int    `json:"task_count"`
	DocumentCount int   `json:"document_count"`
}

type aiSummaryResponse struct {
	Summary   string `json:"summary"`
	ModelUsed string `json:"model_used"`
}

func (h *Handler) fetchAISummary(ctx context.Context, c db.GetCaseByIDRow, taskCount, docCount int) string {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	payload, _ := json.Marshal(aiSummaryRequest{
		CaseNumber:    c.CaseNumber,
		Title:         c.Title,
		Status:        string(c.Status),
		Priority:      string(c.Priority),
		Description:   c.Description,
		TaskCount:     taskCount,
		DocumentCount: docCount,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.aiServiceURL+"/report/summary", bytes.NewReader(payload))
	if err != nil {
		return defaultSummary(c)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return defaultSummary(c)
	}
	defer resp.Body.Close()

	var out aiSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Summary == "" {
		return defaultSummary(c)
	}
	return out.Summary
}

func defaultSummary(c db.GetCaseByIDRow) string {
	return fmt.Sprintf(
		"Case %s (%s) was initiated as a %s priority investigation. "+
			"Current status: %s. "+
			"This report provides a comprehensive record of all investigation activities, "+
			"evidence collected, tasks completed, and communications logged.",
		c.CaseNumber, c.Title, c.Priority, c.Status,
	)
}

// ── PDF builder ───────────────────────────────────────────────────────────────

const (
	marginL  = 15.0
	marginR  = 15.0
	marginT  = 15.0
	pageW    = 210.0 // A4
	contentW = pageW - marginL - marginR
)

func buildPDF(
	c db.GetCaseByIDRow,
	summary string,
	tasks []db.ListTasksByCaseRow,
	timeline []db.ListTimelineEventsRow,
	documents []db.ListDocumentsByCaseRow,
	notes []db.ListNotesByCaseRow,
	comms []db.ListCommLogsRow,
	insights db.CaseAiInsight,
) *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginL, marginT, marginR)
	pdf.SetAutoPageBreak(true, 15)

	pdf.SetTitle(fmt.Sprintf("Case Report — %s", c.CaseNumber), false)
	pdf.SetAuthor("OpsOwl AI", false)
	pdf.SetCreator("OpsOwl Report Engine", false)

	addCoverPage(pdf, c)
	addExecutiveSummary(pdf, summary, c, insights)
	addTimeline(pdf, timeline)
	addDocuments(pdf, documents)
	addTasks(pdf, tasks)
	if len(comms) > 0 {
		addComms(pdf, comms)
	}
	if len(notes) > 0 {
		addNotes(pdf, notes)
	}
	addFooter(pdf, c)

	return pdf
}

// ── Cover page ────────────────────────────────────────────────────────────────

func addCoverPage(pdf *fpdf.Fpdf, c db.GetCaseByIDRow) {
	pdf.AddPage()

	// Dark header band
	pdf.SetFillColor(15, 23, 42)
	pdf.Rect(0, 0, 210, 60, "F")

	// Brand name
	pdf.SetTextColor(34, 211, 238) // cyan-400
	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(marginL, 15)
	pdf.CellFormat(contentW, 10, "OpsOwl AI", "", 0, "L", false, 0, "")

	pdf.SetTextColor(148, 163, 184) // slate-400
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(marginL, 26)
	pdf.CellFormat(contentW, 6, "OPERATIONS INTELLIGENCE WORKSPACE", "", 0, "L", false, 0, "")

	pdf.SetTextColor(241, 245, 249) // slate-100
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(marginL, 45)
	pdf.CellFormat(contentW, 6, "INVESTIGATION CASE REPORT", "", 0, "L", false, 0, "")

	// Case number — big mono
	pdf.SetTextColor(15, 23, 42)
	pdf.SetFont("Courier", "B", 28)
	pdf.SetXY(marginL, 75)
	pdf.SetTextColor(30, 41, 59)
	pdf.CellFormat(contentW, 14, c.CaseNumber, "", 2, "L", false, 0, "")

	// Title
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetX(marginL)
	pdf.MultiCell(contentW, 8, c.Title, "", "L", false)

	pdf.Ln(6)
	drawDivider(pdf)
	pdf.Ln(4)

	// Status / Priority / SLA row
	statusColor := statusBgColor(string(c.Status))
	pdf.SetFillColor(statusColor[0], statusColor[1], statusColor[2])
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetX(marginL)
	pdf.CellFormat(36, 7, "  "+strings.ToUpper(strings.ReplaceAll(string(c.Status), "_", " ")), "", 0, "L", true, 0, "")
	pdf.SetX(pdf.GetX() + 3)

	prioColor := prioBgColor(string(c.Priority))
	pdf.SetFillColor(prioColor[0], prioColor[1], prioColor[2])
	pdf.CellFormat(28, 7, "  "+strings.ToUpper(string(c.Priority)), "", 0, "L", true, 0, "")

	if c.SlaBreached {
		pdf.SetX(pdf.GetX() + 3)
		pdf.SetFillColor(239, 68, 68)
		pdf.CellFormat(28, 7, "  SLA BREACHED", "", 0, "L", true, 0, "")
	}
	pdf.Ln(12)

	// Details grid
	pdf.SetTextColor(15, 23, 42)
	details := [][]string{
		{"Created By", c.CreatedByName + " <" + c.CreatedByEmail + ">"},
		{"Assigned To", nullStrVal(c.AssignedToName, "Unassigned")},
		{"Created At", c.CreatedAt.UTC().Format("02 Jan 2006 15:04 UTC")},
		{"Last Updated", c.UpdatedAt.UTC().Format("02 Jan 2006 15:04 UTC")},
	}
	if c.DueDate.Valid {
		details = append(details, []string{"Due Date", c.DueDate.Time.UTC().Format("02 Jan 2006")})
	}
	if c.SlaBreachAt.Valid {
		details = append(details, []string{"SLA Deadline", c.SlaBreachAt.Time.UTC().Format("02 Jan 2006 15:04 UTC")})
	}
	if len(c.Tags) > 0 {
		details = append(details, []string{"Tags", strings.Join(c.Tags, ", ")})
	}

	for _, row := range details {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetX(marginL)
		pdf.CellFormat(38, 6, row[0]+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(contentW-38, 6, row[1], "", 2, "L", false, 0, "")
	}

	pdf.Ln(8)
	drawDivider(pdf)
	pdf.Ln(4)

	// Description
	if c.Description != "" {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetX(marginL)
		pdf.CellFormat(contentW, 6, "DESCRIPTION", "", 2, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(30, 41, 59)
		pdf.SetX(marginL)
		pdf.MultiCell(contentW, 5, c.Description, "", "L", false)
	}

	// Report generated stamp (bottom)
	pdf.SetY(270)
	drawDivider(pdf)
	pdf.Ln(2)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(148, 163, 184)
	pdf.SetX(marginL)
	pdf.CellFormat(contentW, 5,
		fmt.Sprintf("Report generated by OpsOwl AI on %s UTC — CONFIDENTIAL", time.Now().UTC().Format("02 Jan 2006 15:04")),
		"", 0, "C", false, 0, "")
}

// ── Executive Summary ─────────────────────────────────────────────────────────

func addExecutiveSummary(pdf *fpdf.Fpdf, summary string, c db.GetCaseByIDRow, ins db.CaseAiInsight) {
	pdf.AddPage()
	sectionHeader(pdf, "Executive Summary")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(30, 41, 59)
	pdf.SetX(marginL)
	pdf.MultiCell(contentW, 6, summary, "", "L", false)

	if ins.RiskScore.Valid {
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetX(marginL)
		pdf.CellFormat(contentW, 6, "AI RISK ASSESSMENT", "", 2, "L", false, 0, "")

		var score float64
		fmt.Sscanf(ins.RiskScore.String, "%f", &score)
		level := "Unknown"
		if ins.RiskLevel.Valid {
			level = strings.ToUpper(ins.RiskLevel.String)
		}

		riskColor := riskScoreColor(score)
		pdf.SetFillColor(riskColor[0], riskColor[1], riskColor[2])
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetX(marginL)
		pdf.CellFormat(50, 8, fmt.Sprintf("  Risk Score: %.0f / 100  (%s)", score, level), "", 0, "L", true, 0, "")
		pdf.Ln(12)

		if len(ins.RiskFactors) > 2 {
			var rf []interface{}
			if err := json.Unmarshal(ins.RiskFactors, &rf); err == nil && len(rf) > 0 {
				pdf.SetFont("Helvetica", "B", 9)
				pdf.SetTextColor(71, 85, 105)
				pdf.SetX(marginL)
				pdf.CellFormat(contentW, 6, "TOP RISK FACTORS", "", 2, "L", false, 0, "")
				pdf.SetFont("Helvetica", "", 9)
				pdf.SetTextColor(30, 41, 59)
				for i, f := range rf {
					if i >= 5 {
						break
					}
					pdf.SetX(marginL)
					pdf.CellFormat(contentW, 5, fmt.Sprintf("  • %v", f), "", 2, "L", false, 0, "")
				}
			}
		}
	}
}

// ── Timeline ──────────────────────────────────────────────────────────────────

func addTimeline(pdf *fpdf.Fpdf, events []db.ListTimelineEventsRow) {
	pdf.AddPage()
	sectionHeader(pdf, fmt.Sprintf("Case Timeline  (%d events)", len(events)))

	if len(events) == 0 {
		emptyState(pdf, "No timeline events recorded.")
		return
	}

	tableHeader(pdf, []float64{32, 70, 45, contentW - 147}, []string{"When", "Event", "Actor", "Details"})

	for _, e := range events {
		actor := "System"
		if e.ActorName.Valid {
			actor = e.ActorName.String
		}

		var payload map[string]interface{}
		_ = json.Unmarshal(e.Payload, &payload)
		details := ""
		if v, ok := payload["filename"]; ok {
			details = fmt.Sprintf("File: %v", v)
		} else if v, ok := payload["to"]; ok {
			details = fmt.Sprintf("→ %v", v)
		} else if v, ok := payload["body"]; ok {
			s := fmt.Sprintf("%v", v)
			if len(s) > 50 {
				s = s[:50] + "…"
			}
			details = s
		}

		eventLabel := formatEventType(string(e.EventType))
		row := []string{
			e.OccurredAt.UTC().Format("02 Jan 15:04"),
			eventLabel,
			actor,
			details,
		}
		tableRow(pdf, []float64{32, 70, 45, contentW - 147}, row)
	}
}

// ── Documents ─────────────────────────────────────────────────────────────────

func addDocuments(pdf *fpdf.Fpdf, docs []db.ListDocumentsByCaseRow) {
	pdf.AddPage()
	sectionHeader(pdf, fmt.Sprintf("Evidence & Documents  (%d files)", len(docs)))

	if len(docs) == 0 {
		emptyState(pdf, "No documents uploaded to this case.")
		return
	}

	// Summary counts
	counts := map[string]int{}
	for _, d := range docs {
		counts[string(d.ProcessingStatus)]++
	}
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.SetX(marginL)
	parts := []string{}
	for s, n := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", n, s))
	}
	pdf.CellFormat(contentW, 5, strings.Join(parts, "  ·  "), "", 2, "L", false, 0, "")
	pdf.Ln(4)

	colW := []float64{70, 18, 25, 25, contentW - 138}
	tableHeader(pdf, colW, []string{"Filename", "Type", "Size", "Status", "Uploaded By"})

	for _, d := range docs {
		row := []string{
			truncate(d.OriginalFilename, 38),
			strings.ToUpper(d.FileType),
			formatBytes(d.FileSizeBytes),
			string(d.ProcessingStatus),
			d.UploadedByName,
		}
		tableRow(pdf, colW, row)
	}
}

// ── Tasks ─────────────────────────────────────────────────────────────────────

func addTasks(pdf *fpdf.Fpdf, tasks []db.ListTasksByCaseRow) {
	pdf.AddPage()

	done := 0
	for _, t := range tasks {
		if t.Status == db.TaskStatusDone {
			done++
		}
	}
	sectionHeader(pdf, fmt.Sprintf("Tasks  (%d/%d completed)", done, len(tasks)))

	if len(tasks) == 0 {
		emptyState(pdf, "No tasks created for this case.")
		return
	}

	colW := []float64{80, 20, 18, 35, contentW - 153}
	tableHeader(pdf, colW, []string{"Title", "Priority", "Status", "Assignee", "Due"})

	for _, t := range tasks {
		due := ""
		if t.DueDate.Valid {
			due = t.DueDate.Time.UTC().Format("02 Jan 2006")
		}
		assignee := "Unassigned"
		if t.AssignedToName.Valid {
			assignee = t.AssignedToName.String
		}
		row := []string{
			truncate(t.Title, 42),
			strings.ToUpper(string(t.Priority)),
			string(t.Status),
			assignee,
			due,
		}
		tableRow(pdf, colW, row)
	}
}

// ── Communication Logs ────────────────────────────────────────────────────────

func addComms(pdf *fpdf.Fpdf, comms []db.ListCommLogsRow) {
	pdf.AddPage()
	sectionHeader(pdf, fmt.Sprintf("Communication Log  (%d entries)", len(comms)))

	colW := []float64{28, 20, 40, 55, contentW - 143}
	tableHeader(pdf, colW, []string{"Date", "Channel", "Contact", "Subject/Body", "Logged By"})

	for _, c := range comms {
		subject := ""
		if c.Subject.Valid {
			subject = c.Subject.String
		} else {
			subject = truncate(c.Body, 30)
		}
		row := []string{
			c.OccurredAt.UTC().Format("02 Jan 15:04"),
			string(c.Channel),
			truncate(c.ContactName, 22),
			truncate(subject, 30),
			c.LoggedByName,
		}
		tableRow(pdf, colW, row)
	}
}

// ── Notes ─────────────────────────────────────────────────────────────────────

func addNotes(pdf *fpdf.Fpdf, notes []db.ListNotesByCaseRow) {
	pdf.AddPage()
	sectionHeader(pdf, fmt.Sprintf("Investigation Notes  (%d)", len(notes)))

	for _, n := range notes {
		if pdf.GetY() > 250 {
			pdf.AddPage()
		}
		// Note header
		pdf.SetFillColor(30, 41, 59)
		pdf.SetTextColor(203, 213, 225)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetX(marginL)
		pdf.CellFormat(contentW, 7, fmt.Sprintf("  %s  —  %s  —  %s",
			n.Title,
			n.AuthorName,
			n.CreatedAt.UTC().Format("02 Jan 2006 15:04"),
		), "", 2, "L", true, 0, "")

		// Note body (text-only)
		body := n.ContentText
		if body == "" {
			body = stripHTML(n.Content)
		}
		if len(body) > 600 {
			body = body[:600] + "…"
		}
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(30, 41, 59)
		pdf.SetX(marginL)
		pdf.MultiCell(contentW, 5, body, "", "L", false)
		pdf.Ln(3)
	}
}

// ── Audit footer page ─────────────────────────────────────────────────────────

func addFooter(pdf *fpdf.Fpdf, c db.GetCaseByIDRow) {
	pdf.AddPage()
	sectionHeader(pdf, "Report Footer & Classification")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(30, 41, 59)
	rows := [][]string{
		{"Case Number", c.CaseNumber},
		{"Classification", "CONFIDENTIAL — INTERNAL USE ONLY"},
		{"Report Generated", time.Now().UTC().Format("02 January 2006, 15:04:05 UTC")},
		{"System", "OpsOwl AI — Operations Intelligence Workspace"},
		{"Legal Notice", "This report contains privileged and confidential information. Unauthorised"},
		{"", "disclosure or reproduction is strictly prohibited."},
	}
	for _, r := range rows {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(71, 85, 105)
		pdf.SetX(marginL)
		pdf.CellFormat(48, 6, r[0]+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(30, 41, 59)
		pdf.CellFormat(contentW-48, 6, r[1], "", 2, "L", false, 0, "")
	}

	pdf.Ln(10)
	// Page numbers on all pages
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(148, 163, 184)
		pdf.CellFormat(0, 5,
			fmt.Sprintf("OpsOwl AI Confidential  ·  %s  ·  Page %d", c.CaseNumber, pdf.PageNo()),
			"", 0, "C", false, 0, "")
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func sectionHeader(pdf *fpdf.Fpdf, title string) {
	pdf.SetFillColor(15, 23, 42)
	pdf.SetTextColor(34, 211, 238)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetX(marginL)
	pdf.CellFormat(contentW, 10, "  "+title, "", 2, "L", true, 0, "")
	pdf.Ln(4)
}

func drawDivider(pdf *fpdf.Fpdf) {
	pdf.SetDrawColor(51, 65, 85)
	pdf.SetLineWidth(0.3)
	pdf.Line(marginL, pdf.GetY(), marginL+contentW, pdf.GetY())
}

func emptyState(pdf *fpdf.Fpdf, msg string) {
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(100, 116, 139)
	pdf.SetX(marginL)
	pdf.CellFormat(contentW, 6, msg, "", 2, "L", false, 0, "")
}

func tableHeader(pdf *fpdf.Fpdf, widths []float64, headers []string) {
	pdf.SetFillColor(30, 41, 59)
	pdf.SetTextColor(148, 163, 184)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetX(marginL)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 6, " "+h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
}

func tableRow(pdf *fpdf.Fpdf, widths []float64, cells []string) {
	if pdf.GetY() > 265 {
		pdf.AddPage()
	}
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(30, 41, 59)
	pdf.SetFillColor(241, 245, 249)
	pdf.SetX(marginL)
	for i, cell := range cells {
		pdf.CellFormat(widths[i], 5, " "+cell, "1", 0, "L", false, 0, "")
	}
	pdf.Ln(-1)
}

func statusBgColor(status string) [3]int {
	switch status {
	case "open":           return [3]int{34, 211, 238}
	case "in_progress":    return [3]int{99, 102, 241}
	case "pending_review": return [3]int{245, 158, 11}
	case "escalated":      return [3]int{239, 68, 68}
	case "closed":         return [3]int{74, 222, 128}
	default:               return [3]int{100, 116, 139}
	}
}

func prioBgColor(prio string) [3]int {
	switch prio {
	case "critical": return [3]int{239, 68, 68}
	case "high":     return [3]int{249, 115, 22}
	case "medium":   return [3]int{245, 158, 11}
	default:         return [3]int{100, 116, 139}
	}
}

func riskScoreColor(score float64) [3]int {
	if score >= 75 {
		return [3]int{239, 68, 68}
	}
	if score >= 40 {
		return [3]int{245, 158, 11}
	}
	return [3]int{74, 222, 128}
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func nullStrVal(ns sql.NullString, fallback string) string {
	if ns.Valid {
		return ns.String
	}
	return fallback
}

func formatEventType(et string) string {
	return strings.ReplaceAll(strings.Title(strings.ReplaceAll(et, "_", " ")), "Ai", "AI")
}

func stripHTML(s string) string {
	// Minimal HTML stripping — remove tags, decode common entities
	var b strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
			b.WriteRune(' ')
		} else if !inTag {
			b.WriteRune(r)
		}
	}
	result := b.String()
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&#39;", "'")
	return strings.TrimSpace(result)
}
