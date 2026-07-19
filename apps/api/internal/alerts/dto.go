package alerts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("alert not found")
	ErrSourceNotFound = errors.New("alert source not found")
	ErrRuleNotFound   = errors.New("alert rule not found")
	ErrAlreadyConverted = errors.New("alert has already been converted to a case")
	ErrAlreadyDismissed = errors.New("alert has already been dismissed")
)

// ── Alert types (canonical taxonomy) ─────────────────────────────────────────

var KnownAlertTypes = []string{
	"UNUSUAL_CASH_ACTIVITY",
	"RAPID_MOVEMENT",
	"VELOCITY_BREACH",
	"STRUCTURING",
	"LAYERING",
	"TRADE_BASED_ML",
	"HIGH_RISK_GEOGRAPHY",
	"DORMANT_REACTIVATION",
	"NEW_ACCOUNT_SPIKE",
	"SANCTIONS_HIT",
	"PEP_MATCH",
	"ADVERSE_MEDIA",
	"CTR_THRESHOLD",
	"UNUSUAL_PATTERN",
	"BENEFICIAL_OWNERSHIP",
}

// ── Canonical alert model (normalised from any TMS) ───────────────────────────

type NormalisedAlert struct {
	ExternalAlertID string    `json:"alert_id"`
	AlertType       string    `json:"alert_type"`
	RiskScore       int       `json:"risk_score"` // 0–100
	RiskLevel       string    `json:"risk_level"` // low, medium, high, critical
	SubjectName     string    `json:"subject_name"`
	SubjectIDType   string    `json:"subject_id_type"`
	SubjectIDNumber string    `json:"subject_id_number"`
	AccountNumber   string    `json:"account_number"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	TransactionDate time.Time `json:"transaction_date"`
	Description     string    `json:"description"`
	SourceSystem    string    `json:"source_system"`
}

// ── Source (registered TMS connection) ───────────────────────────────────────

type CreateSourceRequest struct {
	Name          string `json:"name"`
	SourceType    string `json:"source_type"`    // webhook, manual, csv
	SourceSystem  string `json:"source_system"`  // actimize, sas_aml, oracle_fccm, featurespace, generic
	Description   string `json:"description"`
	WebhookSecret string `json:"webhook_secret"` // if non-empty, HMAC-SHA256 enforcement is enabled
}

type UpdateSourceRequest struct {
	Name         *string `json:"name,omitempty"`
	SourceType   *string `json:"source_type,omitempty"`
	SourceSystem *string `json:"source_system,omitempty"`
	Description  *string `json:"description,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

type AlertSourceResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	SourceType    string    `json:"source_type"`
	SourceSystem  string    `json:"source_system"`
	WebhookSecret string    `json:"webhook_secret,omitempty"` // returned only on create
	Description   string    `json:"description"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ── Routing rule ──────────────────────────────────────────────────────────────

type CreateRuleRequest struct {
	Name               string     `json:"name"`
	AlertType          string     `json:"alert_type"`    // '*' for catch-all
	MinRiskScore       int        `json:"min_risk_score"`
	PriorityMapping    string     `json:"priority_mapping"` // critical | high | medium | low
	Tags               []string   `json:"tags"`
	AutoAssignUserID   *uuid.UUID `json:"auto_assign_user_id,omitempty"`
	PlaybookCaseType   string     `json:"playbook_case_type"`
	DedupWindowHours   int        `json:"dedup_window_hours"`
	Mode               string     `json:"mode"`       // queue | auto
	SortOrder          int        `json:"sort_order"`
}

type UpdateRuleRequest struct {
	Name             *string    `json:"name,omitempty"`
	AlertType        *string    `json:"alert_type,omitempty"`
	MinRiskScore     *int       `json:"min_risk_score,omitempty"`
	PriorityMapping  *string    `json:"priority_mapping,omitempty"`
	Tags             []string   `json:"tags,omitempty"`
	AutoAssignUserID *uuid.UUID `json:"auto_assign_user_id,omitempty"`
	PlaybookCaseType *string    `json:"playbook_case_type,omitempty"`
	DedupWindowHours *int       `json:"dedup_window_hours,omitempty"`
	Mode             *string    `json:"mode,omitempty"`
	SortOrder        *int       `json:"sort_order,omitempty"`
	IsActive         *bool      `json:"is_active,omitempty"`
}

type AlertRuleResponse struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	AlertType        string     `json:"alert_type"`
	MinRiskScore     int        `json:"min_risk_score"`
	PriorityMapping  string     `json:"priority_mapping"`
	Tags             []string   `json:"tags"`
	AutoAssignUserID *uuid.UUID `json:"auto_assign_user_id,omitempty"`
	PlaybookCaseType string     `json:"playbook_case_type"`
	DedupWindowHours int        `json:"dedup_window_hours"`
	Mode             string     `json:"mode"`
	SortOrder        int        `json:"sort_order"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ── Incoming alert ────────────────────────────────────────────────────────────

type AlertResponse struct {
	ID                uuid.UUID  `json:"id"`
	ExternalAlertID   string     `json:"external_alert_id"`
	AlertType         string     `json:"alert_type"`
	RiskScore         int        `json:"risk_score"`
	RiskLevel         string     `json:"risk_level"`
	SubjectName       string     `json:"subject_name"`
	SubjectIDType     string     `json:"subject_id_type"`
	SubjectIDNumber   string     `json:"subject_id_number"`
	AccountNumber     string     `json:"account_number"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	TransactionDate   *time.Time `json:"transaction_date,omitempty"`
	Description       string     `json:"description"`
	SourceSystem      string     `json:"source_system"`
	Status            string     `json:"status"`
	LinkedCaseID      *uuid.UUID `json:"linked_case_id,omitempty"`
	LinkedCaseNumber  string     `json:"linked_case_number,omitempty"`
	SubjectID         *uuid.UUID `json:"subject_id,omitempty"`
	MatchedRuleID     *uuid.UUID `json:"matched_rule_id,omitempty"`
	DismissReason     string     `json:"dismiss_reason,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ConvertAlertRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Tags        []string `json:"tags"`
}

type DismissAlertRequest struct {
	Reason string `json:"reason"`
}

type IngestResponse struct {
	AlertID   uuid.UUID `json:"alert_id"`
	Status    string    `json:"status"` // queued, auto_converted, deduplicated
	CaseID    *uuid.UUID `json:"case_id,omitempty"`
	Message   string    `json:"message"`
}

type AlertListFilters struct {
	Status    string
	AlertType string
	RiskLevel string
	SourceID  *uuid.UUID
	Limit     int
	// Cursor-based pagination (preferred). Encodes {created_at, id} of the last seen row.
	// When set, Offset is ignored.
	Cursor *string
	// Offset kept for backwards-compat with any existing callers; ignored when Cursor is set.
	Offset int
}

type alertCursor struct {
	CreatedAt time.Time `json:"t"`
	ID        uuid.UUID `json:"id"`
}

func encodeAlertCursor(createdAt time.Time, id uuid.UUID) string {
	b, _ := json.Marshal(alertCursor{CreatedAt: createdAt, ID: id})
	return base64.StdEncoding.EncodeToString(b)
}

func decodeAlertCursor(s string) (time.Time, uuid.UUID, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	var c alertCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return c.CreatedAt, c.ID, nil
}
