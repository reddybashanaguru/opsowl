package ubo

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByCaseID(ctx context.Context, caseID string) ([]*UBORecord, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, case_id, entity_name, owner_name, owner_nationality,
		       owner_id_type, owner_id_number, owner_birth_date,
		       ownership_pct, control_type, status, registry_url,
		       verified_by::text, verified_at, discrepancy_note,
		       created_at, updated_at
		FROM ubo_records
		WHERE case_id = $1
		ORDER BY created_at ASC`, caseID)
	if err != nil {
		return nil, fmt.Errorf("ubo: list by case: %w", err)
	}
	defer rows.Close()

	var result []*UBORecord
	for rows.Next() {
		rec := &UBORecord{}
		var verifiedBy sql.NullString
		var verifiedAt sql.NullTime
		if err := rows.Scan(
			&rec.ID, &rec.CaseID, &rec.EntityName, &rec.OwnerName,
			&rec.OwnerNationality, &rec.OwnerIDType, &rec.OwnerIDNumber,
			&rec.OwnerBirthDate, &rec.OwnershipPct, &rec.ControlType,
			&rec.Status, &rec.RegistryURL,
			&verifiedBy, &verifiedAt, &rec.DiscrepancyNote,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ubo: scan: %w", err)
		}
		if verifiedBy.Valid {
			rec.VerifiedBy = verifiedBy.String
		}
		if verifiedAt.Valid {
			t := verifiedAt.Time
			rec.VerifiedAt = &t
		}
		result = append(result, rec)
	}
	return result, nil
}

func (r *Repository) Insert(ctx context.Context, rec *UBORecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ubo_records
			(id, case_id, entity_name, owner_name, owner_nationality,
			 owner_id_type, owner_id_number, owner_birth_date,
			 ownership_pct, control_type, status, registry_url,
			 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		rec.ID, rec.CaseID, rec.EntityName, rec.OwnerName,
		rec.OwnerNationality, rec.OwnerIDType, rec.OwnerIDNumber,
		rec.OwnerBirthDate, rec.OwnershipPct, rec.ControlType,
		rec.Status, rec.RegistryURL,
		rec.CreatedAt, rec.UpdatedAt,
	)
	return err
}

type UpdateStatusParams struct {
	ID              string
	Status          VerificationStatus
	VerifiedBy      string
	DiscrepancyNote string
}

func (r *Repository) UpdateStatus(ctx context.Context, p UpdateStatusParams) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE ubo_records
		SET status = $2,
		    verified_by = CASE WHEN $3 = '' THEN verified_by ELSE $3::uuid END,
		    verified_at = CASE WHEN $2 IN ('verified','discrepant') THEN NOW() ELSE verified_at END,
		    discrepancy_note = $4,
		    updated_at = $5
		WHERE id = $1`,
		p.ID, string(p.Status), p.VerifiedBy, p.DiscrepancyNote, time.Now().UTC(),
	)
	return err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM ubo_records WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CountPending(ctx context.Context, caseID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ubo_records WHERE case_id = $1 AND status IN ('pending','discrepant')`,
		caseID,
	).Scan(&n)
	return n, err
}
