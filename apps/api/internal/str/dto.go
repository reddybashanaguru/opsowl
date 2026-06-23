package str

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ── request DTOs ──────────────────────────────────────────────────────────────

type CreateDraftRequest struct {
	ReportType      string          `json:"report_type"`       // STR | SAR | CTR
	SuspicionDate   *string         `json:"suspicion_date"`    // Art. 15 — "YYYY-MM-DD" date string
	SuspicionBasis  string          `json:"suspicion_basis"`
	Narrative       string          `json:"narrative"`
	EvidenceSummary string          `json:"evidence_summary"`
	FilingEntity    json.RawMessage `json:"filing_entity"`
	Subject         json.RawMessage `json:"subject"`
	Transactions    json.RawMessage `json:"transactions"`
}

type ApproveRequest struct {
	// body is empty — approver identity comes from JWT claims
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

type MarkFiledRequest struct {
	FIUReference string `json:"fiu_reference"`
}

// ── response DTOs ─────────────────────────────────────────────────────────────

type FilingResponse struct {
	ID              uuid.UUID       `json:"id"`
	CaseID          uuid.UUID       `json:"case_id"`
	ReportType      string          `json:"report_type"`
	Status          string          `json:"status"`
	SubmittedBy     *uuid.UUID      `json:"submitted_by,omitempty"`
	SubmittedByName string          `json:"submitted_by_name,omitempty"`
	ApprovedBy      *uuid.UUID      `json:"approved_by,omitempty"`
	ApprovedByName  string          `json:"approved_by_name,omitempty"`
	SubmittedAt     *time.Time      `json:"submitted_at,omitempty"`
	ApprovedAt      *time.Time      `json:"approved_at,omitempty"`
	FiledAt         *time.Time      `json:"filed_at,omitempty"`
	FIUReference    string          `json:"fiu_reference,omitempty"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
	SuspicionDate   *time.Time      `json:"suspicion_date,omitempty"`
	SuspicionBasis  string          `json:"suspicion_basis"`
	Narrative       string          `json:"narrative"`
	EvidenceSummary string          `json:"evidence_summary"`
	FilingEntity    json.RawMessage `json:"filing_entity"`
	Subject         json.RawMessage `json:"subject"`
	Transactions    json.RawMessage `json:"transactions"`
	DeadlineDays    int             `json:"deadline_days"`    // days remaining until filing deadline
	IsOverdue       bool            `json:"is_overdue"`       // true when past the regulatory deadline
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
