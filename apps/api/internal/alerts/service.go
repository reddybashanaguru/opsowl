package alerts

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/reddybashanaguru/opsowl/internal/auth"
)

var defaultOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// CaseCreator is implemented by an adapter in server.go to avoid import cycles.
type CaseCreator interface {
	CreateFromAlert(ctx context.Context, p AlertCaseParams) (caseID uuid.UUID, caseNumber string, err error)
}

// AlertCaseParams carries the data needed to auto-create a case from an alert.
type AlertCaseParams struct {
	Title       string
	Description string
	Priority    string
	Tags        []string
	AssignTo    *uuid.UUID
	CaseType    string
	AlertID     uuid.UUID
	SubjectID   *uuid.UUID
}

// SubjectResolver is implemented by an adapter in server.go.
type SubjectResolver interface {
	ResolveOrCreate(ctx context.Context, idType, idNumber, name string) (uuid.UUID, error)
}

// InsightsTrigger queues AI risk analysis immediately after a case is auto-created.
// Implemented by an adapter in server.go; the triggeredBy actor is captured inside.
type InsightsTrigger interface {
	TriggerAnalysis(caseID uuid.UUID)
}

type Service struct {
	repo            *Repository
	caseMgr         CaseCreator
	subjectMgr      SubjectResolver
	insightsTrigger InsightsTrigger
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) WithCaseCreator(c CaseCreator) *Service {
	s.caseMgr = c
	return s
}

func (s *Service) WithSubjectResolver(sr SubjectResolver) *Service {
	s.subjectMgr = sr
	return s
}

func (s *Service) WithInsightsTrigger(t InsightsTrigger) *Service {
	s.insightsTrigger = t
	return s
}

// VerifyWebhookSignature checks the HMAC-SHA256 signature of an inbound payload.
// The header format is "sha256=<lowercase_hex>" (same convention as GitHub/Stripe).
func VerifyWebhookSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), want)
}

// IngestAlert normalises a raw TMS payload, applies deduplication, runs rule
// matching, persists the alert, and auto-creates a case when the matched rule
// is in "auto" mode.
func (s *Service) IngestAlert(ctx context.Context, sourceID uuid.UUID, raw []byte) (*IngestResponse, error) {
	src, err := s.repo.GetSourceByID(ctx, sourceID, defaultOrgID)
	if err != nil {
		return nil, err
	}
	if !src.IsActive {
		return nil, fmt.Errorf("source is inactive")
	}

	alert, err := Normalize(src.SourceSystem, raw)
	if err != nil {
		return nil, fmt.Errorf("normalize: %w", err)
	}

	// Load active rules ordered by sort_order asc
	rules, err := s.repo.ListRules(ctx, defaultOrgID)
	if err != nil {
		return nil, err
	}
	matched := matchRule(rules, alert)

	// Semantic dedup — same entity + alert_type within the rule's window
	dedupHours := 72
	if matched.rule.ID != uuid.Nil && matched.rule.DedupWindowHours > 0 {
		dedupHours = matched.rule.DedupWindowHours
	}
	dup, err := s.repo.FindDuplicate(ctx, defaultOrgID,
		alert.SubjectIDType, alert.SubjectIDNumber, alert.AlertType, dedupHours)
	if err != nil {
		return nil, err
	}
	if dup != nil {
		return &IngestResponse{
			AlertID: dup.ID,
			Status:  "deduplicated",
			CaseID:  dup.LinkedCaseID,
			Message: "duplicate alert for the same entity within dedup window",
		}, nil
	}

	// Persist (ON CONFLICT on external_alert_id updates updated_at and returns existing)
	stored, err := s.repo.Ingest(ctx, defaultOrgID, &sourceID, alert, raw)
	if err != nil {
		return nil, err
	}

	// Non-pending means the external_alert_id already existed and was processed
	if stored.Status != "pending" {
		return &IngestResponse{
			AlertID: stored.ID,
			Status:  "deduplicated",
			CaseID:  stored.LinkedCaseID,
			Message: "duplicate external_alert_id — already processed",
		}, nil
	}

	// Attach matched rule
	if matched.rule.ID != uuid.Nil {
		_ = s.repo.SetMatchedRule(ctx, stored.ID, matched.rule.ID)
	}

	// Resolve or create subject (best-effort; alert is stored regardless)
	var subjectID *uuid.UUID
	if s.subjectMgr != nil && alert.SubjectIDNumber != "" {
		sid, serr := s.subjectMgr.ResolveOrCreate(ctx,
			alert.SubjectIDType, alert.SubjectIDNumber, alert.SubjectName)
		if serr == nil {
			subjectID = &sid
			_ = s.repo.SetStatus(ctx, stored.ID, "pending", map[string]any{"subject_id": sid})
		}
	}

	// Auto-create case when the matched rule mode is "auto"
	if matched.mode == "auto" && s.caseMgr != nil {
		title := formatAlertCaseTitle(alert.AlertType, alert.SubjectName, alert.AccountNumber)
		desc := alert.Description
		if desc == "" {
			desc = fmt.Sprintf("Auto-escalated by rule. Source: %s. Risk score: %d (%s).",
				alert.SourceSystem, alert.RiskScore, alert.RiskLevel)
		}
		caseID, _, cerr := s.caseMgr.CreateFromAlert(ctx, AlertCaseParams{
			Title:       title,
			Description: desc,
			Priority:    matched.priority,
			Tags:        matched.tags,
			AssignTo:    matched.rule.AutoAssignUserID,
			CaseType:    matched.rule.PlaybookCaseType,
			AlertID:     stored.ID,
			SubjectID:   subjectID,
		})
		if cerr == nil {
			_ = s.repo.SetStatus(ctx, stored.ID, "converted", map[string]any{
				"linked_case_id": caseID,
				"converted_at":   time.Now(),
			})
			if s.insightsTrigger != nil {
				s.insightsTrigger.TriggerAnalysis(caseID)
			}
			return &IngestResponse{
				AlertID: stored.ID,
				Status:  "auto_converted",
				CaseID:  &caseID,
				Message: fmt.Sprintf("auto-converted by rule: %s", matched.rule.Name),
			}, nil
		}
	}

	return &IngestResponse{
		AlertID: stored.ID,
		Status:  "queued",
		Message: "alert queued for analyst review",
	}, nil
}

// ConvertToCase manually converts an alert to a case (analyst-initiated).
func (s *Service) ConvertToCase(ctx context.Context, claims *auth.Claims, alertID uuid.UUID, req ConvertAlertRequest) (*IngestResponse, error) {
	alert, err := s.repo.GetByID(ctx, alertID, defaultOrgID)
	if err != nil {
		return nil, err
	}
	if alert.Status == "converted" {
		return nil, ErrAlreadyConverted
	}
	if alert.Status == "dismissed" {
		return nil, ErrAlreadyDismissed
	}
	if s.caseMgr == nil {
		return nil, fmt.Errorf("case creation not available")
	}

	title := req.Title
	if title == "" {
		title = fmt.Sprintf("[%s] %s", alert.AlertType, alert.SubjectName)
	}
	priority := req.Priority
	if priority == "" {
		priority = alert.RiskLevel
		if priority == "" {
			priority = "medium"
		}
	}

	caseID, _, err := s.caseMgr.CreateFromAlert(ctx, AlertCaseParams{
		Title:       title,
		Description: req.Description,
		Priority:    priority,
		Tags:        req.Tags,
		AssignTo:    &claims.UserID,
		AlertID:     alertID,
		SubjectID:   alert.SubjectID,
	})
	if err != nil {
		return nil, fmt.Errorf("create case: %w", err)
	}

	_ = s.repo.SetStatus(ctx, alertID, "converted", map[string]any{
		"linked_case_id": caseID,
		"converted_by":   claims.UserID,
		"converted_at":   time.Now(),
	})

	return &IngestResponse{
		AlertID: alertID,
		Status:  "converted",
		CaseID:  &caseID,
		Message: "alert converted to case",
	}, nil
}

// DismissAlert marks an alert as dismissed with an analyst-provided reason.
func (s *Service) DismissAlert(ctx context.Context, claims *auth.Claims, alertID uuid.UUID, req DismissAlertRequest) error {
	alert, err := s.repo.GetByID(ctx, alertID, defaultOrgID)
	if err != nil {
		return err
	}
	if alert.Status == "dismissed" {
		return ErrAlreadyDismissed
	}
	if alert.Status == "converted" {
		return ErrAlreadyConverted
	}
	reason := req.Reason
	if reason == "" {
		reason = "no reason provided"
	}
	return s.repo.SetStatus(ctx, alertID, "dismissed", map[string]any{
		"dismiss_reason": reason,
		"dismissed_by":   claims.UserID,
		"dismissed_at":   time.Now(),
	})
}

// ── Bulk operations ───────────────────────────────────────────────────────────

const maxBulkAlerts = 100

// BulkAction represents a single result item from a bulk operation.
type BulkResultItem struct {
	AlertID uuid.UUID `json:"alert_id"`
	Error   string    `json:"error,omitempty"`
}

// BulkResult summarises the outcome of a bulk operation.
type BulkResult struct {
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Errors    []BulkResultItem `json:"errors,omitempty"`
}

// BulkDismiss dismisses up to maxBulkAlerts alerts in a single call.
// Each alert is processed independently — a failure on one does not block
// the others.  Results include per-item errors for the caller to surface.
func (s *Service) BulkDismiss(ctx context.Context, claims *auth.Claims, alertIDs []uuid.UUID, reason string) (*BulkResult, error) {
	if len(alertIDs) == 0 {
		return &BulkResult{}, nil
	}
	if len(alertIDs) > maxBulkAlerts {
		return nil, fmt.Errorf("bulk dismiss limited to %d alerts per request", maxBulkAlerts)
	}
	if reason == "" {
		reason = "bulk dismissed by analyst"
	}

	result := &BulkResult{}
	for _, id := range alertIDs {
		err := s.DismissAlert(ctx, claims, id, DismissAlertRequest{Reason: reason})
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BulkResultItem{AlertID: id, Error: err.Error()})
		} else {
			result.Succeeded++
		}
	}
	return result, nil
}

// BulkMarkDuplicate marks up to maxBulkAlerts alerts as duplicate in a single
// call.  Same independent-processing semantics as BulkDismiss.
func (s *Service) BulkMarkDuplicate(ctx context.Context, claims *auth.Claims, alertIDs []uuid.UUID, reason string) (*BulkResult, error) {
	if len(alertIDs) == 0 {
		return &BulkResult{}, nil
	}
	if len(alertIDs) > maxBulkAlerts {
		return nil, fmt.Errorf("bulk mark-duplicate limited to %d alerts per request", maxBulkAlerts)
	}
	if reason == "" {
		reason = "bulk marked duplicate by analyst"
	}

	result := &BulkResult{}
	for _, id := range alertIDs {
		alert, err := s.repo.GetByID(ctx, id, defaultOrgID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BulkResultItem{AlertID: id, Error: "alert not found"})
			continue
		}
		if alert.Status == "converted" || alert.Status == "dismissed" || alert.Status == "duplicate" {
			result.Failed++
			result.Errors = append(result.Errors, BulkResultItem{AlertID: id, Error: "alert already actioned: " + alert.Status})
			continue
		}
		setErr := s.repo.SetStatus(ctx, id, "duplicate", map[string]any{
			"dismiss_reason": reason,
			"dismissed_by":   claims.UserID,
			"dismissed_at":   time.Now(),
		})
		if setErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, BulkResultItem{AlertID: id, Error: setErr.Error()})
		} else {
			result.Succeeded++
		}
	}
	return result, nil
}

// ── Source CRUD ───────────────────────────────────────────────────────────────

func (s *Service) ListSources(ctx context.Context) ([]AlertSourceResponse, error) {
	return s.repo.ListSources(ctx, defaultOrgID)
}

func (s *Service) CreateSource(ctx context.Context, claims *auth.Claims, req CreateSourceRequest) (*AlertSourceResponse, error) {
	return s.repo.CreateSource(ctx, defaultOrgID, claims.UserID, req)
}

func (s *Service) GetSource(ctx context.Context, id uuid.UUID) (*AlertSourceResponse, error) {
	return s.repo.GetSourceByID(ctx, id, defaultOrgID)
}

func (s *Service) UpdateSource(ctx context.Context, id uuid.UUID, req UpdateSourceRequest) (*AlertSourceResponse, error) {
	return s.repo.UpdateSource(ctx, id, defaultOrgID, req)
}

func (s *Service) DeleteSource(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSource(ctx, id, defaultOrgID)
}

// ── Rule CRUD ─────────────────────────────────────────────────────────────────

func (s *Service) ListRules(ctx context.Context) ([]AlertRuleResponse, error) {
	return s.repo.ListRules(ctx, defaultOrgID)
}

func (s *Service) CreateRule(ctx context.Context, claims *auth.Claims, req CreateRuleRequest) (*AlertRuleResponse, error) {
	return s.repo.CreateRule(ctx, defaultOrgID, claims.UserID, req)
}

func (s *Service) UpdateRule(ctx context.Context, id uuid.UUID, req UpdateRuleRequest) (*AlertRuleResponse, error) {
	return s.repo.UpdateRule(ctx, id, defaultOrgID, req)
}

func (s *Service) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRule(ctx, id, defaultOrgID)
}

// ── Alert queries ─────────────────────────────────────────────────────────────

func (s *Service) List(ctx context.Context, f AlertListFilters) ([]AlertResponse, int64, error) {
	return s.repo.List(ctx, defaultOrgID, f)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*AlertResponse, error) {
	return s.repo.GetByID(ctx, id, defaultOrgID)
}

// formatAlertCaseTitle builds a human-readable case title from the alert type,
// subject name, and optional account number. Example:
//
//	"Structuring Transaction – Rahul Sharma (AC-7890123)"
func formatAlertCaseTitle(alertType, subjectName, accountNumber string) string {
	parts := strings.Split(strings.ToLower(alertType), "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	title := strings.Join(parts, " ") + " – " + subjectName
	if accountNumber != "" {
		title += " (" + accountNumber + ")"
	}
	return title
}
