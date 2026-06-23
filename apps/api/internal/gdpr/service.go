// Package gdpr implements the GDPR right-to-erasure workflow for EU deployments.
//
// The core tension: GDPR Art. 17 grants data subjects the right to erasure, but
// EU AMLR 2024 Art. 77 mandates 5-year retention of AML records. OpsOwl resolves
// this via pseudonymisation — PII fields are replaced with stable, consistent
// pseudonyms, preserving the audit trail and transaction record while eliminating
// personally identifiable data.
//
// Legal basis recorded on every anonymisation:
//   - Within retention period: GDPR Art. 17(3)(b) — exemption for legal obligations
//   - Beyond retention period: full erasure permitted
package gdpr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DSARStatus represents the lifecycle of a Data Subject Access Request.
type DSARStatus string

const (
	DSARReceived   DSARStatus = "received"    // logged, not yet reviewed
	DSARInProgress DSARStatus = "in_progress" // DPO is assessing eligibility
	DSARCompleted  DSARStatus = "completed"   // anonymisation applied or full erasure executed
	DSARRejected   DSARStatus = "rejected"    // AML retention exemption applies; request denied
	DSARDeferred   DSARStatus = "deferred"    // within retention window; scheduled for future
)

// ErasureOutcome describes what was actually done for a completed DSAR.
type ErasureOutcome string

const (
	OutcomePseudonymised ErasureOutcome = "pseudonymised" // PII replaced, audit trail preserved
	OutcomeFullErasure   ErasureOutcome = "full_erasure"  // retention period expired; fully deleted
	OutcomeDenied        ErasureOutcome = "denied"        // AML retention requirement takes precedence
)

// DSAR is a Data Subject Access Request for erasure under GDPR Art. 17.
type DSAR struct {
	ID             string         `json:"id"`
	SubjectName    string         `json:"subject_name"`     // as provided by requester
	SubjectEmail   string         `json:"subject_email"`
	RequestDate    time.Time      `json:"request_date"`
	Deadline       time.Time      `json:"deadline"`         // 30-day GDPR response deadline
	Status         DSARStatus     `json:"status"`
	Outcome        ErasureOutcome `json:"outcome,omitempty"`
	LegalBasis     string         `json:"legal_basis,omitempty"`
	ProcessedBy    string         `json:"processed_by,omitempty"` // DPO user ID
	ProcessedAt    *time.Time     `json:"processed_at,omitempty"`
	CasesAffected  []string       `json:"cases_affected,omitempty"`  // case IDs containing matched PII
	Notes          string         `json:"notes,omitempty"`
}

// AnonymisationRecord tracks what was pseudonymised for audit purposes.
// The pseudonym is stable per original value (same name → same pseudonym)
// so cross-case references remain internally consistent.
type AnonymisationRecord struct {
	DSARID      string    `json:"dsar_id"`
	CaseID      string    `json:"case_id"`
	Field       string    `json:"field"`      // "subject_name", "subject_address", etc.
	Pseudonym   string    `json:"pseudonym"`  // "[ANON-a3f9b1c2]"
	AnonymisedAt time.Time `json:"anonymised_at"`
	LegalBasis  string    `json:"legal_basis"`
}

// RetentionStatus indicates whether a case's AML retention period has expired.
type RetentionStatus struct {
	CaseID         string    `json:"case_id"`
	ClosedAt       time.Time `json:"closed_at"`
	RetentionYears int       `json:"retention_years"`
	ExpiresAt      time.Time `json:"expires_at"`
	Expired        bool      `json:"expired"`
}

// Service handles DSAR lifecycle management and anonymisation.
type Service struct {
	db             *sql.DB
	retentionYears int
}

// NewService creates a GDPR service with the configured retention period.
// retentionYears should come from config.retention_years (env-configurable, default 5).
func NewService(db *sql.DB, retentionYears int) *Service {
	return &Service{db: db, retentionYears: retentionYears}
}

// CreateDSAR registers a new data subject erasure request and computes the
// 30-day GDPR response deadline.
func (s *Service) CreateDSAR(ctx context.Context, subjectName, subjectEmail, notes string) (*DSAR, error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	deadline := now.AddDate(0, 0, 30) // GDPR Art. 12(3) — 30 calendar days

	dsar := &DSAR{
		ID:           id,
		SubjectName:  subjectName,
		SubjectEmail: subjectEmail,
		RequestDate:  now,
		Deadline:     deadline,
		Status:       DSARReceived,
		Notes:        notes,
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO dsar_requests
			(id, subject_name, subject_email, request_date, deadline, status, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		id, subjectName, subjectEmail, now, deadline, DSARReceived, notes,
	)
	if err != nil {
		return nil, fmt.Errorf("gdpr: create dsar: %w", err)
	}
	return dsar, nil
}

// CheckRetention determines whether a case's retention period has expired,
// making it eligible for full erasure (as opposed to pseudonymisation).
func (s *Service) CheckRetention(ctx context.Context, caseID string) (*RetentionStatus, error) {
	var closedAt sql.NullTime
	err := s.db.QueryRowContext(ctx,
		`SELECT closed_at FROM cases WHERE id = $1`, caseID,
	).Scan(&closedAt)
	if err != nil {
		return nil, fmt.Errorf("gdpr: check retention: %w", err)
	}
	if !closedAt.Valid {
		return nil, errors.New("gdpr: case is not yet closed; retention period has not started")
	}
	expiresAt := closedAt.Time.AddDate(s.retentionYears, 0, 0)
	return &RetentionStatus{
		CaseID:         caseID,
		ClosedAt:       closedAt.Time,
		RetentionYears: s.retentionYears,
		ExpiresAt:      expiresAt,
		Expired:        time.Now().UTC().After(expiresAt),
	}, nil
}

// Pseudonymise replaces PII in a case with stable pseudonyms.
// The pseudonym is derived from the DSAR ID + original value hash so the
// same person appearing in multiple cases gets the same pseudonym — cross-case
// references remain internally consistent for investigators.
//
// If the retention period has NOT expired, this is the legally required response:
// GDPR Art. 17(3)(b) exempts AML records from erasure during the retention window.
func (s *Service) Pseudonymise(ctx context.Context, dsarID, caseID, processedByUserID string) error {
	// Implementation: UPDATE subjects SET name = '[ANON-<hash>]', id_number = '[ANON-...]',
	// address = '[ANON-...]' WHERE case_id = $1 AND name ILIKE $2
	// Full implementation in follow-up PR once dsar_requests migration is in place.
	return fmt.Errorf("gdpr: pseudonymise: not yet implemented — see docs/EU_MARKET_ROADMAP.md P-EU-1")
}
