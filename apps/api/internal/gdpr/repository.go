package gdpr

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]*DSAR, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, subject_name, subject_email, request_date, deadline,
		       status, outcome, legal_basis,
		       processed_by::text, processed_at, notes, created_at, updated_at
		FROM dsar_requests
		ORDER BY deadline ASC`)
	if err != nil {
		return nil, fmt.Errorf("gdpr: list: %w", err)
	}
	defer rows.Close()

	var result []*DSAR
	for rows.Next() {
		d := &DSAR{}
		var outcome, legalBasis, processedBy sql.NullString
		var processedAt sql.NullTime
		if err := rows.Scan(
			&d.ID, &d.SubjectName, &d.SubjectEmail,
			&d.RequestDate, &d.Deadline,
			&d.Status, &outcome, &legalBasis,
			&processedBy, &processedAt, &d.Notes,
			&d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if outcome.Valid {
			d.Outcome = ErasureOutcome(outcome.String)
		}
		if legalBasis.Valid {
			d.LegalBasis = legalBasis.String
		}
		if processedBy.Valid {
			d.ProcessedBy = processedBy.String
		}
		if processedAt.Valid {
			t := processedAt.Time
			d.ProcessedAt = &t
		}
		result = append(result, d)
	}
	return result, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*DSAR, error) {
	d := &DSAR{}
	var outcome, legalBasis, processedBy sql.NullString
	var processedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, subject_name, subject_email, request_date, deadline,
		       status, outcome, legal_basis,
		       processed_by::text, processed_at, notes, created_at, updated_at
		FROM dsar_requests WHERE id = $1`, id,
	).Scan(
		&d.ID, &d.SubjectName, &d.SubjectEmail,
		&d.RequestDate, &d.Deadline,
		&d.Status, &outcome, &legalBasis,
		&processedBy, &processedAt, &d.Notes,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	if outcome.Valid {
		d.Outcome = ErasureOutcome(outcome.String)
	}
	if legalBasis.Valid {
		d.LegalBasis = legalBasis.String
	}
	if processedBy.Valid {
		d.ProcessedBy = processedBy.String
	}
	if processedAt.Valid {
		t := processedAt.Time
		d.ProcessedAt = &t
	}
	return d, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status, legalBasis, byUserID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE dsar_requests SET status = $2, legal_basis = $3,
		       processed_by = CASE WHEN $4 = '' THEN processed_by ELSE $4::uuid END,
		       updated_at = $5
		WHERE id = $1`,
		id, status, legalBasis, byUserID, time.Now().UTC(),
	)
	return err
}

func (r *Repository) ListAnonymisationRecords(ctx context.Context, dsarID string) ([]AnonymisationRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT dsar_id, subject_id::text, fields_masked, pseudonym, legal_basis, anonymised_at
		FROM anonymisation_records
		WHERE dsar_id = $1
		ORDER BY anonymised_at ASC`, dsarID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AnonymisationRecord
	for rows.Next() {
		var rec AnonymisationRecord
		if err := rows.Scan(&rec.DSARID, &rec.SubjectID, pq.Array(&rec.FieldsMasked), &rec.Pseudonym, &rec.LegalBasis, &rec.AnonymisedAt); err != nil {
			return nil, err
		}
		result = append(result, rec)
	}
	return result, nil
}
