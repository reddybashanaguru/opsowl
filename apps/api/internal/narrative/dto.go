package narrative

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("narrative not found")
	ErrForbidden         = errors.New("forbidden")
	ErrBadRequest        = errors.New("bad request")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrAlreadyApproved   = errors.New("narrative is already approved")
)

// ── Narrative types ───────────────────────────────────────────────────────────

const (
	TypeInvestigationSummary = "investigation_summary"
	TypeSTRSuspicionBasis    = "str_suspicion_basis"
	TypeSARNarrative         = "sar_narrative"
	TypeClosureSummary       = "closure_summary"
)

var validTypes = map[string]bool{
	TypeInvestigationSummary: true,
	TypeSTRSuspicionBasis:    true,
	TypeSARNarrative:         true,
	TypeClosureSummary:       true,
}

// ── Request types ─────────────────────────────────────────────────────────────

type GenerateRequest struct {
	NarrativeType     string `json:"narrative_type"` // defaults to investigation_summary
	InvestigatorNotes string `json:"investigator_notes,omitempty"`
}

type EditRequest struct {
	Body string `json:"body"`
}

type ReviewRequest struct {
	Reason string `json:"reason,omitempty"` // required for rejection
}

// ── Response types ────────────────────────────────────────────────────────────

type UserRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type NarrativeResponse struct {
	ID               string    `json:"id"`
	CaseID           string    `json:"case_id"`
	NarrativeType    string    `json:"narrative_type"`
	Status           string    `json:"status"`
	Body             string    `json:"body"`
	BodyEdited       *string   `json:"body_edited,omitempty"`
	ActiveBody       string    `json:"active_body"` // body_edited if set, else body
	GeneratedBy      *UserRef  `json:"generated_by,omitempty"`
	ReviewedBy       *UserRef  `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason  string    `json:"rejection_reason,omitempty"`
	ModelUsed        string    `json:"model_used"`
	PromptVersion    string    `json:"prompt_version"`
	FindingsIncluded int       `json:"findings_included"`
	TxnsIncluded     int       `json:"txns_included"`
	AILogID          *string   `json:"ai_log_id,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ── AI service wire types ─────────────────────────────────────────────────────

type aiNarrativeFinding struct {
	Title           string `json:"title"`
	Category        string `json:"category"`
	Severity        string `json:"severity"`
	Description     string `json:"description"`
	NarrativeWeight string `json:"narrative_weight"`
}

type aiNarrativeTransaction struct {
	TxnRef     string  `json:"txn_ref"`
	TxnDate    string  `json:"txn_date"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	TxnType    string  `json:"txn_type"`
	Channel    string  `json:"channel"`
	FromName   string  `json:"from_name"`
	ToName     string  `json:"to_name"`
	Narration  string  `json:"narration"`
	IsFlagged  bool    `json:"is_flagged"`
	FlagReason string  `json:"flag_reason"`
}

type aiSubjectProfile struct {
	Name      string `json:"name"`
	IDType    string `json:"id_type,omitempty"`
	IDNumber  string `json:"id_number,omitempty"`
	KYCTier   string `json:"kyc_tier,omitempty"`
	RiskLevel string `json:"risk_level,omitempty"`
	IsPEP     bool   `json:"is_pep"`
	Notes     string `json:"notes,omitempty"`
}

type aiInsightContext struct {
	RiskScore          float64  `json:"risk_score"`
	RiskFactors        []string `json:"risk_factors"`
	Anomalies          []string `json:"anomalies"`
	RecommendedActions []string `json:"recommended_actions"`
}

type aiNarrativeRequest struct {
	CaseID            string                   `json:"case_id"`
	CaseNumber        string                   `json:"case_number"`
	CaseTitle         string                   `json:"case_title"`
	CaseDescription   string                   `json:"case_description"`
	RiskLevel         string                   `json:"risk_level"`
	NarrativeType     string                   `json:"narrative_type"`
	Findings          []aiNarrativeFinding     `json:"findings"`
	Transactions      []aiNarrativeTransaction `json:"transactions"`
	SubjectNames      []string                 `json:"subject_names"`
	SubjectProfiles   []aiSubjectProfile       `json:"subject_profiles,omitempty"`
	AIInsights        *aiInsightContext         `json:"ai_insights,omitempty"`
	InvestigatorNotes string                   `json:"investigator_notes"`
}

type aiNarrativeResponse struct {
	Narrative     string `json:"narrative"`
	ModelUsed     string `json:"model_used"`
	FindingsUsed  int    `json:"findings_used"`
	TxnsUsed      int    `json:"txns_used"`
	PromptVersion string `json:"prompt_version"`
}
