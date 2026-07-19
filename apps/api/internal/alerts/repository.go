package alerts

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ── Alert Sources ─────────────────────────────────────────────────────────────

func (r *Repository) CreateSource(ctx context.Context, orgID, createdBy uuid.UUID, req CreateSourceRequest) (*AlertSourceResponse, error) {
	secret := req.WebhookSecret // empty string = no HMAC enforcement
	var src AlertSourceResponse
	err := r.db.QueryRowContext(ctx, `
INSERT INTO alert_sources (org_id, name, source_type, source_system, webhook_secret, description, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7)
RETURNING id, name, source_type, source_system, webhook_secret, description, is_active, created_at, updated_at`,
		orgID, req.Name, req.SourceType, req.SourceSystem, secret, req.Description, createdBy,
	).Scan(&src.ID, &src.Name, &src.SourceType, &src.SourceSystem, &src.WebhookSecret,
		&src.Description, &src.IsActive, &src.CreatedAt, &src.UpdatedAt)
	return &src, err
}

func (r *Repository) ListSources(ctx context.Context, orgID uuid.UUID) ([]AlertSourceResponse, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, source_type, source_system, '', description, is_active, created_at, updated_at
         FROM alert_sources WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertSourceResponse
	for rows.Next() {
		var s AlertSourceResponse
		if err := rows.Scan(&s.ID, &s.Name, &s.SourceType, &s.SourceSystem, &s.WebhookSecret,
			&s.Description, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []AlertSourceResponse{}
	}
	return out, rows.Err()
}

func (r *Repository) GetSourceByID(ctx context.Context, id, orgID uuid.UUID) (*AlertSourceResponse, error) {
	var s AlertSourceResponse
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, source_type, source_system, webhook_secret, description, is_active, created_at, updated_at
         FROM alert_sources WHERE id=$1 AND org_id=$2`, id, orgID,
	).Scan(&s.ID, &s.Name, &s.SourceType, &s.SourceSystem, &s.WebhookSecret,
		&s.Description, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrSourceNotFound
	}
	return &s, err
}

func (r *Repository) UpdateSource(ctx context.Context, id, orgID uuid.UUID, req UpdateSourceRequest) (*AlertSourceResponse, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1
	set := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}
	if req.Name != nil {
		set("name", *req.Name)
	}
	if req.SourceType != nil {
		set("source_type", *req.SourceType)
	}
	if req.SourceSystem != nil {
		set("source_system", *req.SourceSystem)
	}
	if req.Description != nil {
		set("description", *req.Description)
	}
	if req.IsActive != nil {
		set("is_active", *req.IsActive)
	}
	args = append(args, id, orgID)
	_, err := r.db.ExecContext(ctx,
		"UPDATE alert_sources SET "+strings.Join(sets, ", ")+
			fmt.Sprintf(" WHERE id=$%d AND org_id=$%d", n, n+1), args...)
	if err != nil {
		return nil, err
	}
	return r.GetSourceByID(ctx, id, orgID)
}

func (r *Repository) DeleteSource(ctx context.Context, id, orgID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM alert_sources WHERE id=$1 AND org_id=$2", id, orgID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrSourceNotFound
	}
	return nil
}

// ── Alert Rules ───────────────────────────────────────────────────────────────

func (r *Repository) CreateRule(ctx context.Context, orgID, createdBy uuid.UUID, req CreateRuleRequest) (*AlertRuleResponse, error) {
	if req.DedupWindowHours == 0 {
		req.DedupWindowHours = 72
	}
	if req.Mode == "" {
		req.Mode = "queue"
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	var rule AlertRuleResponse
	err := r.db.QueryRowContext(ctx, `
INSERT INTO alert_rules (org_id, name, alert_type, min_risk_score, priority_mapping, tags,
  auto_assign_user_id, playbook_case_type, dedup_window_hours, mode, sort_order, created_by)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING id, name, alert_type, min_risk_score, priority_mapping, tags,
  auto_assign_user_id, playbook_case_type, dedup_window_hours, mode, sort_order, is_active, created_at, updated_at`,
		orgID, req.Name, req.AlertType, req.MinRiskScore, req.PriorityMapping,
		pq.Array(req.Tags), req.AutoAssignUserID, req.PlaybookCaseType,
		req.DedupWindowHours, req.Mode, req.SortOrder, createdBy,
	).Scan(&rule.ID, &rule.Name, &rule.AlertType, &rule.MinRiskScore, &rule.PriorityMapping,
		(*pq.StringArray)(&rule.Tags), &rule.AutoAssignUserID, &rule.PlaybookCaseType,
		&rule.DedupWindowHours, &rule.Mode, &rule.SortOrder, &rule.IsActive,
		&rule.CreatedAt, &rule.UpdatedAt)
	if rule.Tags == nil {
		rule.Tags = []string{}
	}
	return &rule, err
}

func (r *Repository) ListRules(ctx context.Context, orgID uuid.UUID) ([]AlertRuleResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, alert_type, min_risk_score, priority_mapping, tags,
  auto_assign_user_id, playbook_case_type, dedup_window_hours, mode, sort_order, is_active, created_at, updated_at
FROM alert_rules WHERE org_id=$1 ORDER BY sort_order, created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertRuleResponse
	for rows.Next() {
		var rule AlertRuleResponse
		var tags pq.StringArray
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.AlertType, &rule.MinRiskScore,
			&rule.PriorityMapping, &tags, &rule.AutoAssignUserID, &rule.PlaybookCaseType,
			&rule.DedupWindowHours, &rule.Mode, &rule.SortOrder, &rule.IsActive,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Tags = []string(tags)
		if rule.Tags == nil {
			rule.Tags = []string{}
		}
		out = append(out, rule)
	}
	if out == nil {
		out = []AlertRuleResponse{}
	}
	return out, rows.Err()
}

func (r *Repository) UpdateRule(ctx context.Context, id, orgID uuid.UUID, req UpdateRuleRequest) (*AlertRuleResponse, error) {
	sets := []string{"updated_at = NOW()"}
	args := []any{}
	n := 1
	set := func(col string, val any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, n))
		args = append(args, val)
		n++
	}
	if req.Name != nil {
		set("name", *req.Name)
	}
	if req.AlertType != nil {
		set("alert_type", *req.AlertType)
	}
	if req.MinRiskScore != nil {
		set("min_risk_score", *req.MinRiskScore)
	}
	if req.PriorityMapping != nil {
		set("priority_mapping", *req.PriorityMapping)
	}
	if req.Tags != nil {
		set("tags", pq.Array(req.Tags))
	}
	if req.AutoAssignUserID != nil {
		set("auto_assign_user_id", *req.AutoAssignUserID)
	}
	if req.PlaybookCaseType != nil {
		set("playbook_case_type", *req.PlaybookCaseType)
	}
	if req.DedupWindowHours != nil {
		set("dedup_window_hours", *req.DedupWindowHours)
	}
	if req.Mode != nil {
		set("mode", *req.Mode)
	}
	if req.SortOrder != nil {
		set("sort_order", *req.SortOrder)
	}
	if req.IsActive != nil {
		set("is_active", *req.IsActive)
	}
	args = append(args, id, orgID)
	_, err := r.db.ExecContext(ctx,
		"UPDATE alert_rules SET "+strings.Join(sets, ", ")+
			fmt.Sprintf(" WHERE id=$%d AND org_id=$%d", n, n+1), args...)
	if err != nil {
		return nil, err
	}
	// Return updated
	var rule AlertRuleResponse
	var tags pq.StringArray
	err = r.db.QueryRowContext(ctx, `
SELECT id, name, alert_type, min_risk_score, priority_mapping, tags,
  auto_assign_user_id, playbook_case_type, dedup_window_hours, mode, sort_order, is_active, created_at, updated_at
FROM alert_rules WHERE id=$1`, id).Scan(
		&rule.ID, &rule.Name, &rule.AlertType, &rule.MinRiskScore,
		&rule.PriorityMapping, &tags, &rule.AutoAssignUserID, &rule.PlaybookCaseType,
		&rule.DedupWindowHours, &rule.Mode, &rule.SortOrder, &rule.IsActive,
		&rule.CreatedAt, &rule.UpdatedAt)
	rule.Tags = []string(tags)
	if rule.Tags == nil {
		rule.Tags = []string{}
	}
	return &rule, err
}

func (r *Repository) DeleteRule(ctx context.Context, id, orgID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, "DELETE FROM alert_rules WHERE id=$1 AND org_id=$2", id, orgID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRuleNotFound
	}
	return nil
}

// ── Incoming Alerts ───────────────────────────────────────────────────────────

func (r *Repository) Ingest(ctx context.Context, orgID uuid.UUID, sourceID *uuid.UUID, alert NormalisedAlert, raw []byte) (*AlertResponse, error) {
	rawJSON := json.RawMessage(raw)
	var txDate *time.Time
	if !alert.TransactionDate.IsZero() {
		txDate = &alert.TransactionDate
	}
	a := &AlertResponse{}
	var linkedCaseID, subjectID, matchedRuleID, dismissedBy *uuid.UUID
	var dismissReason sql.NullString
	var linkedCaseNumber sql.NullString

	err := r.db.QueryRowContext(ctx, `
INSERT INTO incoming_alerts
  (org_id, source_id, external_alert_id, alert_type, risk_score, risk_level,
   subject_name, subject_id_type, subject_id_number, account_number,
   amount, currency, transaction_date, description, source_system, raw_payload)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (org_id, source_id, external_alert_id)
  WHERE external_alert_id != ''
DO UPDATE SET updated_at = NOW()
RETURNING id, external_alert_id, alert_type, risk_score, risk_level,
  subject_name, subject_id_type, subject_id_number, account_number,
  amount, currency, transaction_date, description, source_system, status,
  linked_case_id, '', subject_id, matched_rule_id,
  dismissed_by, dismiss_reason, created_at, updated_at`,
		orgID, sourceID, alert.ExternalAlertID, alert.AlertType, alert.RiskScore, alert.RiskLevel,
		alert.SubjectName, alert.SubjectIDType, alert.SubjectIDNumber, alert.AccountNumber,
		alert.Amount, alert.Currency, txDate, alert.Description, alert.SourceSystem, rawJSON,
	).Scan(&a.ID, &a.ExternalAlertID, &a.AlertType, &a.RiskScore, &a.RiskLevel,
		&a.SubjectName, &a.SubjectIDType, &a.SubjectIDNumber, &a.AccountNumber,
		&a.Amount, &a.Currency, &a.TransactionDate, &a.Description, &a.SourceSystem, &a.Status,
		&linkedCaseID, &linkedCaseNumber, &subjectID, &matchedRuleID,
		&dismissedBy, &dismissReason, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	a.LinkedCaseID = linkedCaseID
	a.SubjectID = subjectID
	a.MatchedRuleID = matchedRuleID
	if dismissReason.Valid {
		a.DismissReason = dismissReason.String
	}
	return a, nil
}

func (r *Repository) FindDuplicate(ctx context.Context, orgID uuid.UUID, idType, idNumber, alertType string, windowHours int) (*AlertResponse, error) {
	if idNumber == "" || windowHours == 0 {
		return nil, nil
	}
	var a AlertResponse
	var linkedCaseID *uuid.UUID
	err := r.db.QueryRowContext(ctx, `
SELECT id, alert_type, risk_score, risk_level, subject_name, status, linked_case_id, created_at
FROM incoming_alerts
WHERE org_id=$1
  AND subject_id_type=$2 AND subject_id_number=$3
  AND alert_type=$4
  AND status NOT IN ('dismissed')
  AND created_at > NOW() - make_interval(hours => $5)
ORDER BY created_at DESC LIMIT 1`,
		orgID, idType, idNumber, alertType, windowHours,
	).Scan(&a.ID, &a.AlertType, &a.RiskScore, &a.RiskLevel, &a.SubjectName, &a.Status, &linkedCaseID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.LinkedCaseID = linkedCaseID
	return &a, nil
}

func (r *Repository) SetMatchedRule(ctx context.Context, alertID, ruleID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "UPDATE incoming_alerts SET matched_rule_id=$1 WHERE id=$2", ruleID, alertID)
	return err
}

func (r *Repository) SetStatus(ctx context.Context, alertID uuid.UUID, status string, extra map[string]any) error {
	sets := []string{"status = $1", "updated_at = NOW()"}
	args := []any{status}
	n := 2
	for k, v := range extra {
		sets = append(sets, fmt.Sprintf("%s = $%d", k, n))
		args = append(args, v)
		n++
	}
	args = append(args, alertID)
	_, err := r.db.ExecContext(ctx,
		"UPDATE incoming_alerts SET "+strings.Join(sets, ", ")+fmt.Sprintf(" WHERE id=$%d", n), args...)
	return err
}

func (r *Repository) List(ctx context.Context, orgID uuid.UUID, f AlertListFilters) ([]AlertResponse, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 25
	}
	args := []any{orgID}
	where := []string{"ia.org_id = $1"}
	n := 2

	if f.Status != "" {
		where = append(where, fmt.Sprintf("ia.status = $%d", n))
		args = append(args, f.Status)
		n++
	}
	if f.AlertType != "" {
		where = append(where, fmt.Sprintf("ia.alert_type = $%d", n))
		args = append(args, f.AlertType)
		n++
	}
	if f.RiskLevel != "" {
		where = append(where, fmt.Sprintf("ia.risk_level = $%d", n))
		args = append(args, f.RiskLevel)
		n++
	}
	if f.SourceID != nil {
		where = append(where, fmt.Sprintf("ia.source_id = $%d", n))
		args = append(args, *f.SourceID)
		n++
	}

	cond := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM incoming_alerts ia LEFT JOIN cases c ON c.id = ia.linked_case_id WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Cursor-based keyset pagination: avoids slow OFFSET scans at large page numbers.
	// Falls back to OFFSET when no cursor is provided (first page).
	selectQ := `
SELECT ia.id, ia.external_alert_id, ia.alert_type, ia.risk_score, ia.risk_level,
  ia.subject_name, ia.subject_id_type, ia.subject_id_number, ia.account_number,
  ia.amount, ia.currency, ia.transaction_date, ia.description, ia.source_system,
  ia.status, ia.linked_case_id, COALESCE(c.case_number,''), ia.subject_id, ia.matched_rule_id,
  ia.dismiss_reason, ia.created_at, ia.updated_at
FROM incoming_alerts ia
LEFT JOIN cases c ON c.id = ia.linked_case_id
WHERE `
	var rows *sql.Rows
	var err error
	if f.Cursor != nil {
		cursorTime, cursorID, decErr := decodeAlertCursor(*f.Cursor)
		if decErr == nil {
			args = append(args, cursorTime, cursorID, f.Limit)
			rows, err = r.db.QueryContext(ctx, selectQ+cond+
				fmt.Sprintf(` AND (ia.created_at, ia.id) < ($%d, $%d) ORDER BY ia.created_at DESC, ia.id DESC LIMIT $%d`, n, n+1, n+2), args...)
		}
	}
	if rows == nil && err == nil {
		args = append(args, f.Limit, f.Offset)
		rows, err = r.db.QueryContext(ctx, selectQ+cond+
			fmt.Sprintf(` ORDER BY ia.created_at DESC, ia.id DESC LIMIT $%d OFFSET $%d`, n, n+1), args...)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []AlertResponse
	for rows.Next() {
		var a AlertResponse
		var txDate sql.NullTime
		var dismissReason sql.NullString
		if err := rows.Scan(&a.ID, &a.ExternalAlertID, &a.AlertType, &a.RiskScore, &a.RiskLevel,
			&a.SubjectName, &a.SubjectIDType, &a.SubjectIDNumber, &a.AccountNumber,
			&a.Amount, &a.Currency, &txDate, &a.Description, &a.SourceSystem,
			&a.Status, &a.LinkedCaseID, &a.LinkedCaseNumber, &a.SubjectID, &a.MatchedRuleID,
			&dismissReason, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if txDate.Valid {
			a.TransactionDate = &txDate.Time
		}
		if dismissReason.Valid {
			a.DismissReason = dismissReason.String
		}
		out = append(out, a)
	}
	if out == nil {
		out = []AlertResponse{}
	}
	return out, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id, orgID uuid.UUID) (*AlertResponse, error) {
	var a AlertResponse
	var txDate sql.NullTime
	var dismissReason sql.NullString
	err := r.db.QueryRowContext(ctx, `
SELECT ia.id, ia.external_alert_id, ia.alert_type, ia.risk_score, ia.risk_level,
  ia.subject_name, ia.subject_id_type, ia.subject_id_number, ia.account_number,
  ia.amount, ia.currency, ia.transaction_date, ia.description, ia.source_system,
  ia.status, ia.linked_case_id, COALESCE(c.case_number,''), ia.subject_id, ia.matched_rule_id,
  ia.dismiss_reason, ia.created_at, ia.updated_at
FROM incoming_alerts ia
LEFT JOIN cases c ON c.id = ia.linked_case_id
WHERE ia.id=$1 AND ia.org_id=$2`, id, orgID).Scan(
		&a.ID, &a.ExternalAlertID, &a.AlertType, &a.RiskScore, &a.RiskLevel,
		&a.SubjectName, &a.SubjectIDType, &a.SubjectIDNumber, &a.AccountNumber,
		&a.Amount, &a.Currency, &txDate, &a.Description, &a.SourceSystem,
		&a.Status, &a.LinkedCaseID, &a.LinkedCaseNumber, &a.SubjectID, &a.MatchedRuleID,
		&dismissReason, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if txDate.Valid {
		a.TransactionDate = &txDate.Time
	}
	if dismissReason.Valid {
		a.DismissReason = dismissReason.String
	}
	return &a, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func generateSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
