package subjects

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound        = errors.New("subject not found")
	ErrDuplicateID     = errors.New("a subject with this ID type and number already exists")
	ErrForbidden       = errors.New("forbidden")
	ErrNameRequired    = errors.New("canonical name is required")
	ErrAlreadyLinked   = errors.New("subject already linked to this case")
	ErrNotLinked       = errors.New("subject is not linked to this case")
	ErrCaseClosed      = errors.New("case is closed — subjects cannot be linked to a closed case")
)

// --- Request types ---

type CreateSubjectRequest struct {
	CanonicalName      string          `json:"canonical_name"`
	EntityType         string          `json:"entity_type"` // individual, company, account, unknown
	IDType             string          `json:"id_type"`
	IDNumber           string          `json:"id_number"`
	SecondaryIDs       []SecondaryID   `json:"secondary_ids"`
	Nationality        string          `json:"nationality"`
	DateOfBirth        *string         `json:"date_of_birth"` // YYYY-MM-DD
	Gender             string          `json:"gender"`
	Addresses          []Address       `json:"addresses"`
	Phones             []string        `json:"phones"`
	Emails             []string        `json:"emails"`
	IsPEP              bool            `json:"is_pep"`
	IsSanctioned       bool            `json:"is_sanctioned"`
	IsWatchlisted      bool            `json:"is_watchlisted"`
	KYCTier            string          `json:"kyc_tier"`
	Notes              string          `json:"notes"`
}

type UpdateSubjectRequest struct {
	CanonicalName      *string         `json:"canonical_name,omitempty"`
	EntityType         *string         `json:"entity_type,omitempty"`
	IDType             *string         `json:"id_type,omitempty"`
	IDNumber           *string         `json:"id_number,omitempty"`
	SecondaryIDs       []SecondaryID   `json:"secondary_ids,omitempty"`
	Nationality        *string         `json:"nationality,omitempty"`
	DateOfBirth        *string         `json:"date_of_birth,omitempty"`
	Gender             *string         `json:"gender,omitempty"`
	Addresses          []Address       `json:"addresses,omitempty"`
	Phones             []string        `json:"phones,omitempty"`
	Emails             []string        `json:"emails,omitempty"`
	IsPEP              *bool           `json:"is_pep,omitempty"`
	IsSanctioned       *bool           `json:"is_sanctioned,omitempty"`
	IsWatchlisted      *bool           `json:"is_watchlisted,omitempty"`
	KYCTier            *string         `json:"kyc_tier,omitempty"`
	Notes              *string         `json:"notes,omitempty"`
	RiskScore          *int            `json:"risk_score,omitempty"`
	RiskLevel          *string         `json:"risk_level,omitempty"`
	RiskFactors        []string        `json:"risk_factors,omitempty"`
}

type LinkSubjectRequest struct {
	SubjectID uuid.UUID `json:"subject_id"`
	Role      string    `json:"role"` // primary, secondary, counterparty, witness
}

type CreateSubjectAndLinkRequest struct {
	Subject CreateSubjectRequest `json:"subject"`
	Role    string               `json:"role"`
}

// --- Sub-types ---

type SecondaryID struct {
	Type     string `json:"type"`
	Number   string `json:"number"`
	Country  string `json:"country"`
	IssuedBy string `json:"issued_by"`
}

type Address struct {
	Line1     string `json:"line1"`
	Line2     string `json:"line2"`
	City      string `json:"city"`
	State     string `json:"state"`
	Country   string `json:"country"`
	ZIP       string `json:"zip"`
	IsPrimary bool   `json:"is_primary"`
}

// --- Response types ---

type SubjectResponse struct {
	ID              uuid.UUID       `json:"id"`
	CanonicalName   string          `json:"canonical_name"`
	EntityType      string          `json:"entity_type"`
	IDType          string          `json:"id_type"`
	IDNumber        string          `json:"id_number"`
	SecondaryIDs    []SecondaryID   `json:"secondary_ids"`
	Nationality     string          `json:"nationality"`
	DateOfBirth     *string         `json:"date_of_birth,omitempty"`
	Gender          string          `json:"gender"`
	Addresses       []Address       `json:"addresses"`
	Phones          []string        `json:"phones"`
	Emails          []string        `json:"emails"`
	RiskScore       int             `json:"risk_score"`
	RiskLevel       string          `json:"risk_level"`
	RiskFactors     []string        `json:"risk_factors"`
	IsPEP           bool            `json:"is_pep"`
	IsSanctioned    bool            `json:"is_sanctioned"`
	IsWatchlisted   bool            `json:"is_watchlisted"`
	KYCTier         string          `json:"kyc_tier"`
	KYCLastReviewed *time.Time      `json:"kyc_last_reviewed_at,omitempty"`
	TotalCases      int             `json:"total_cases"`
	OpenCases       int             `json:"open_cases"`
	TotalSTRFilings int             `json:"total_str_filings"`
	Notes           string          `json:"notes"`
	FirstSeenAt     time.Time       `json:"first_seen_at"`
	LastSeenAt      time.Time       `json:"last_seen_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CaseSubjectRef struct {
	SubjectID     uuid.UUID `json:"subject_id"`
	CanonicalName string    `json:"canonical_name"`
	IDType        string    `json:"id_type"`
	IDNumber      string    `json:"id_number"`
	EntityType    string    `json:"entity_type"`
	RiskLevel     string    `json:"risk_level"`
	RiskScore     int       `json:"risk_score"`
	IsPEP         bool      `json:"is_pep"`
	IsSanctioned  bool      `json:"is_sanctioned"`
	Role          string    `json:"role"`
	LinkedAt      time.Time `json:"linked_at"`
}

type RiskHistoryEntry struct {
	ID        uuid.UUID `json:"id"`
	RiskScore int       `json:"risk_score"`
	RiskLevel string    `json:"risk_level"`
	Reason    string    `json:"reason"`
	ChangedBy string    `json:"changed_by"`
	CreatedAt time.Time `json:"created_at"`
}

type SubjectCaseRef struct {
	CaseID     uuid.UUID  `json:"case_id"`
	CaseNumber string     `json:"case_number"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Priority   string     `json:"priority"`
	Role       string     `json:"role"`
	LinkedAt   time.Time  `json:"linked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ListFilters struct {
	Search       string
	EntityType   string
	RiskLevel    string
	IsPEP        *bool
	IsSanctioned *bool
	Watchlisted  *bool
	Limit        int
	Offset       int
}
