package scheduled_reports

import (
	"fmt"
	"strings"
	"time"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type ReportData struct {
	Schedule   *Schedule
	Stats      db.DashboardStats
	NewCases   int64
	PeriodDesc string // "last 24 hours" | "last 7 days"
	GeneratedAt time.Time
}

// ComposeEmail builds plain-text and HTML bodies for the summary email.
func ComposeEmail(d ReportData) (subject, plainText, htmlBody string) {
	ts := d.GeneratedAt.UTC().Format("02 Jan 2006")
	subject = fmt.Sprintf("[OpsOwl] %s — %s", d.Schedule.Name, ts)

	// ── Plain text ────────────────────────────────────────────────────────────
	var pt strings.Builder
	pt.WriteString(fmt.Sprintf("OpsOwl Investigation Workspace — %s\n", d.Schedule.Name))
	pt.WriteString(fmt.Sprintf("Report generated: %s UTC\n", d.GeneratedAt.UTC().Format("02 Jan 2006 15:04")))
	pt.WriteString(fmt.Sprintf("Period: %s\n\n", d.PeriodDesc))

	pt.WriteString("── CASE SUMMARY ──────────────────────────────\n")
	pt.WriteString(fmt.Sprintf("New cases (%s): %d\n", d.PeriodDesc, d.NewCases))
	pt.WriteString(fmt.Sprintf("Open:            %d\n", d.Stats.StatusOpen))
	pt.WriteString(fmt.Sprintf("In Progress:     %d\n", d.Stats.StatusInProgress))
	pt.WriteString(fmt.Sprintf("Pending Review:  %d\n", d.Stats.StatusPendingReview))
	pt.WriteString(fmt.Sprintf("Escalated:       %d\n", d.Stats.StatusEscalated))
	pt.WriteString(fmt.Sprintf("Closed:          %d\n\n", d.Stats.StatusClosed))

	pt.WriteString("── BY PRIORITY (active) ──────────────────────\n")
	pt.WriteString(fmt.Sprintf("Critical:  %d\n", d.Stats.PrioCritical))
	pt.WriteString(fmt.Sprintf("High:      %d\n", d.Stats.PrioHigh))
	pt.WriteString(fmt.Sprintf("Medium:    %d\n", d.Stats.PrioMedium))
	pt.WriteString(fmt.Sprintf("Low:       %d\n\n", d.Stats.PrioLow))

	if d.Stats.SLABreached > 0 {
		pt.WriteString(fmt.Sprintf("⚠ SLA BREACHED: %d active cases have breached SLA.\n\n", d.Stats.SLABreached))
	}

	pt.WriteString("────────────────────────────────────────────────\n")
	pt.WriteString("This is an automated report from OpsOwl AI.\n")
	pt.WriteString("Log in to investigate: http://localhost:3001/dashboard\n")
	plainText = pt.String()

	// ── HTML ──────────────────────────────────────────────────────────────────
	htmlBody = buildHTML(d)
	return
}

func buildHTML(d ReportData) string {
	ts := d.GeneratedAt.UTC().Format("02 Jan 2006 15:04 UTC")

	slaAlert := ""
	if d.Stats.SLABreached > 0 {
		slaAlert = fmt.Sprintf(`
      <tr>
        <td colspan="2" style="padding:12px 20px;background:#450a0a;border-left:4px solid #ef4444;border-radius:4px;margin-bottom:12px;">
          <span style="color:#fca5a5;font-weight:700;">⚠ SLA BREACHED:</span>
          <span style="color:#fecaca;"> %d active case(s) have exceeded their SLA deadline.</span>
        </td>
      </tr>`, d.Stats.SLABreached)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>OpsOwl Report</title></head>
<body style="margin:0;padding:0;background:#0f172a;font-family:system-ui,sans-serif;">
<table width="100%%" cellpadding="0" cellspacing="0" style="background:#0f172a;">
  <tr><td align="center" style="padding:32px 16px;">
    <table width="600" cellpadding="0" cellspacing="0" style="background:#1e293b;border-radius:12px;border:1px solid #334155;overflow:hidden;">

      <!-- Header -->
      <tr>
        <td colspan="2" style="background:#0f172a;padding:24px 28px;">
          <span style="color:#22d3ee;font-size:22px;font-weight:800;letter-spacing:-0.5px;">OpsOwl AI</span>
          <span style="color:#64748b;font-size:11px;display:block;margin-top:2px;letter-spacing:1px;">OPERATIONS INTELLIGENCE WORKSPACE</span>
        </td>
      </tr>

      <!-- Title -->
      <tr>
        <td colspan="2" style="padding:20px 28px 8px;">
          <h1 style="margin:0;color:#f1f5f9;font-size:18px;font-weight:700;">%s</h1>
          <p style="margin:4px 0 0;color:#64748b;font-size:12px;">Generated %s &nbsp;·&nbsp; %s</p>
        </td>
      </tr>

      <!-- SLA Alert (if any) -->
      %s

      <!-- New cases highlight -->
      <tr>
        <td colspan="2" style="padding:16px 28px 0;">
          <table width="100%%" cellpadding="0" cellspacing="0" style="background:#164e63;border-radius:8px;padding:16px;">
            <tr>
              <td style="color:#a5f3fc;font-size:13px;font-weight:600;">New cases (%s)</td>
              <td align="right" style="color:#67e8f9;font-size:28px;font-weight:800;">%d</td>
            </tr>
          </table>
        </td>
      </tr>

      <!-- Status breakdown -->
      <tr>
        <td colspan="2" style="padding:20px 28px 8px;">
          <p style="margin:0 0 12px;color:#94a3b8;font-size:11px;font-weight:600;letter-spacing:1px;text-transform:uppercase;">Case Status</p>
          <table width="100%%" cellpadding="0" cellspacing="0">
            %s
          </table>
        </td>
      </tr>

      <!-- Priority breakdown -->
      <tr>
        <td colspan="2" style="padding:8px 28px 20px;">
          <p style="margin:0 0 12px;color:#94a3b8;font-size:11px;font-weight:600;letter-spacing:1px;text-transform:uppercase;">Active by Priority</p>
          <table width="100%%" cellpadding="0" cellspacing="0">
            %s
          </table>
        </td>
      </tr>

      <!-- Footer -->
      <tr>
        <td colspan="2" style="background:#0f172a;padding:16px 28px;border-top:1px solid #1e293b;">
          <p style="margin:0;color:#475569;font-size:11px;text-align:center;">
            Automated report from OpsOwl AI &nbsp;·&nbsp; CONFIDENTIAL — INTERNAL USE ONLY
          </p>
        </td>
      </tr>

    </table>
  </td></tr>
</table>
</body>
</html>`,
		d.Schedule.Name, ts, d.PeriodDesc,
		slaAlert,
		d.PeriodDesc, d.NewCases,
		statusRows(d.Stats),
		priorityRows(d.Stats),
	)
}

func statusRows(s db.DashboardStats) string {
	items := []struct {
		label string
		count int64
		color string
	}{
		{"Open", s.StatusOpen, "#22d3ee"},
		{"In Progress", s.StatusInProgress, "#818cf8"},
		{"Pending Review", s.StatusPendingReview, "#fbbf24"},
		{"Escalated", s.StatusEscalated, "#f87171"},
		{"Closed", s.StatusClosed, "#4ade80"},
	}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(fmt.Sprintf(`<tr>
          <td style="color:#cbd5e1;font-size:13px;padding:4px 0;">%s</td>
          <td align="right" style="color:%s;font-size:16px;font-weight:700;padding:4px 0;">%d</td>
        </tr>`, item.label, item.color, item.count))
	}
	return sb.String()
}

func priorityRows(s db.DashboardStats) string {
	items := []struct {
		label string
		count int64
		color string
	}{
		{"Critical", s.PrioCritical, "#f87171"},
		{"High", s.PrioHigh, "#fb923c"},
		{"Medium", s.PrioMedium, "#fbbf24"},
		{"Low", s.PrioLow, "#94a3b8"},
	}
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(fmt.Sprintf(`<tr>
          <td style="color:#cbd5e1;font-size:13px;padding:4px 0;">%s</td>
          <td align="right" style="color:%s;font-size:16px;font-weight:700;padding:4px 0;">%d</td>
        </tr>`, item.label, item.color, item.count))
	}
	return sb.String()
}
