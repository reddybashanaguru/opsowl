package closure

import (
	"time"

	"github.com/google/uuid"
)

// ── Request types ─────────────────────────────────────────────────────────────

type CreateClosureRequest struct {
	Disposition         string  `json:"disposition"`
	Summary             string  `json:"summary"`
	ReasonCode          string  `json:"reason_code"`
	FiledSTR            bool    `json:"filed_str"`
	STRFileID           *string `json:"str_id"`
	RiskRatingAtClose   string  `json:"risk_rating_at_close"`
	SARConsidered       bool    `json:"sar_considered"`
	SARRejectionReason  string  `json:"sar_rejection_reason"`
}

type UpdateClosureRequest struct {
	Disposition         *string `json:"disposition"`
	Summary             *string `json:"summary"`
	ReasonCode          *string `json:"reason_code"`
	FiledSTR            *bool   `json:"filed_str"`
	STRFileID           *string `json:"str_id"`
	RiskRatingAtClose   *string `json:"risk_rating_at_close"`
	SARConsidered       *bool   `json:"sar_considered"`
	SARRejectionReason  *string `json:"sar_rejection_reason"`
}

type ReviewClosureRequest struct {
	Reason string `json:"reason"`
}

type ReopenRequest struct {
	Reason string `json:"reason"`
}

// ── Checklist types ───────────────────────────────────────────────────────────

type ChecklistItem struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Blocking bool   `json:"blocking"`
	Detail   string `json:"detail"`
}

type ChecklistResult struct {
	CanProceed bool            `json:"can_proceed"`
	Checks     []ChecklistItem `json:"checks"`
}

// ── Response types ────────────────────────────────────────────────────────────

type UserRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type ClosureResponse struct {
	ID                  string     `json:"id"`
	CaseID              string     `json:"case_id"`
	Disposition         string     `json:"disposition"`
	Summary             string     `json:"summary"`
	ReasonCode          string     `json:"reason_code"`
	FiledSTR            bool       `json:"filed_str"`
	STRFileID           *string    `json:"str_id,omitempty"`
	RiskRatingAtClose   string     `json:"risk_rating_at_close"`
	SARConsidered       bool       `json:"sar_considered"`
	SARRejectionReason  string     `json:"sar_rejection_reason,omitempty"`
	Status              string     `json:"status"`
	PreparedBy          UserRef    `json:"prepared_by"`
	CreatedBy           UserRef    `json:"created_by"`
	ReviewedBy          *UserRef   `json:"reviewed_by,omitempty"`
	ApprovedBy          *UserRef   `json:"approved_by,omitempty"`
	RejectedBy          *UserRef   `json:"rejected_by,omitempty"`
	RejectionReason     string     `json:"rejection_reason,omitempty"`
	RejectedAt          *time.Time `json:"rejected_at,omitempty"`
	SubmittedAt         *time.Time `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time `json:"approved_at,omitempty"`
	VoidedAt            *time.Time `json:"voided_at,omitempty"`
	VoidedReason        string     `json:"voided_reason,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
