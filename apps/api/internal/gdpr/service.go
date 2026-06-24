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
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
	CasesAffected  []string       `json:"cases_affected,omitempty"`
	Notes          string         `json:"notes,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// AnonymisationRecord tracks what was pseudonymised for audit purposes.
// The pseudonym is stable per (dsarID, subjectID) so the same subject
// appearing in multiple cases gets a consistent pseudonym — audit cross-references remain intact.
type AnonymisationRecord struct {
	DSARID       string    `json:"dsar_id"`
	SubjectID    string    `json:"subject_id"`
	FieldsMasked []string  `json:"fields_masked"`
	Pseudonym    string    `json:"pseudonym"`
	LegalBasis   string    `json:"legal_basis"`
	AnonymisedAt time.Time `json:"anonymised_at"`
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

// ProcessDSAR finds subjects matching the requester's name and pseudonymises their PII.
// Within the AML retention window: pseudonymise (GDPR Art. 17(3)(b) exemption).
// After retention expires: full soft-delete of the subject record.
//
// The pseudonym is stable per (dsarID, subjectID) pair so the same person appearing
// in multiple cases gets the same pseudonym — audit trail cross-references remain intact.
func (s *Service) ProcessDSAR(ctx context.Context, dsarID, processedByUserID string) error {
	// Load DSAR to get subject name
	var subjectName string
	err := s.db.QueryRowContext(ctx,
		`SELECT subject_name FROM dsar_requests WHERE id = $1`, dsarID,
	).Scan(&subjectName)
	if err == sql.ErrNoRows {
		return errors.New("gdpr: DSAR not found")
	}
	if err != nil {
		return fmt.Errorf("gdpr: load dsar: %w", err)
	}

	// Find all subjects with a matching name (org-wide)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM subjects WHERE canonical_name ILIKE $1 AND deleted_at IS NULL`,
		subjectName,
	)
	if err != nil {
		return fmt.Errorf("gdpr: find subjects: %w", err)
	}
	defer rows.Close()

	var subjectIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		subjectIDs = append(subjectIDs, id)
	}

	for _, subjectID := range subjectIDs {
		pseudonym := stablePseudonym(dsarID, subjectID)

		// Check retention: if any linked case is still within the retention window,
		// pseudonymise rather than delete (GDPR Art. 17(3)(b)).
		var openCases int
		_ = s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM case_subjects WHERE subject_id = $1
			 AND EXISTS (
			   SELECT 1 FROM cases c WHERE c.id = case_subjects.case_id
			   AND (c.closed_at IS NULL OR c.closed_at > $2)
			 )`,
			subjectID, time.Now().UTC().AddDate(-s.retentionYears, 0, 0),
		).Scan(&openCases)

		if openCases > 0 {
			// Within retention — pseudonymise PII fields only
			_, err = s.db.ExecContext(ctx, `
				UPDATE subjects SET
					canonical_name  = $2,
					id_number       = $2,
					secondary_ids   = '[]'::jsonb,
					addresses       = '[]'::jsonb,
					phones          = '{}',
					emails          = '{}',
					date_of_birth   = NULL,
					notes           = '',
					updated_at      = NOW()
				WHERE id = $1`,
				subjectID, pseudonym,
			)
			legalBasis := fmt.Sprintf("GDPR Art. 17(3)(b) — AML retention exemption; %d-year window active. Pseudonym: %s", s.retentionYears, pseudonym)
			s.insertAnonymisationRecord(ctx, dsarID, subjectID, processedByUserID, pseudonym, legalBasis) //nolint:errcheck
		} else {
			// Retention expired — full soft-delete
			_, err = s.db.ExecContext(ctx,
				`UPDATE subjects SET deleted_at = NOW() WHERE id = $1`, subjectID)
			legalBasis := fmt.Sprintf("GDPR Art. 17(1) — retention period of %d years expired; full erasure applied.", s.retentionYears)
			s.insertAnonymisationRecord(ctx, dsarID, subjectID, processedByUserID, "[DELETED]", legalBasis) //nolint:errcheck
		}
		if err != nil {
			return fmt.Errorf("gdpr: pseudonymise subject %s: %w", subjectID, err)
		}
	}

	outcome := ErasureOutcome(OutcomePseudonymised)
	if len(subjectIDs) == 0 {
		outcome = OutcomeDenied
	}
	legalBasis := "GDPR Art. 17 processed"

	_, err = s.db.ExecContext(ctx, `
		UPDATE dsar_requests SET
			status       = 'completed',
			outcome      = $2,
			legal_basis  = $3,
			processed_by = $4::uuid,
			processed_at = NOW(),
			updated_at   = NOW()
		WHERE id = $1`,
		dsarID, string(outcome), legalBasis, processedByUserID,
	)
	return err
}

// stablePseudonym generates a stable 8-char hex pseudonym from dsarID + subjectID.
func stablePseudonym(dsarID, subjectID string) string {
	h := sha256.Sum256([]byte(dsarID + ":" + subjectID))
	return fmt.Sprintf("[ANON-%x]", h[:4])
}

func (s *Service) insertAnonymisationRecord(ctx context.Context, dsarID, subjectID, byUserID, pseudonym, legalBasis string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO anonymisation_records
			(dsar_id, subject_id, fields_masked, pseudonym, legal_basis, anonymised_by)
		VALUES ($1, $2, $3, $4, $5, $6::uuid)`,
		dsarID, subjectID,
		pq.Array([]string{"canonical_name", "id_number", "secondary_ids", "addresses", "phones", "emails", "date_of_birth"}),
		pseudonym, legalBasis, byUserID,
	)
	return err
}
