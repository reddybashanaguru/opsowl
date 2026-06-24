package sanctions

import (
	"context"
	"crypto/sha256"
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

// UpsertEntries bulk-upserts entries and updates the list metadata.
func (r *Repository) UpsertEntries(ctx context.Context, source ListSource, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO sanctions_entries
			(list_source, external_id, entity_type, primary_name, aliases,
			 nationality, birth_date, listed_on, programs, remarks)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (list_source, external_id) DO UPDATE SET
			entity_type  = EXCLUDED.entity_type,
			primary_name = EXCLUDED.primary_name,
			aliases      = EXCLUDED.aliases,
			nationality  = EXCLUDED.nationality,
			birth_date   = EXCLUDED.birth_date,
			listed_on    = EXCLUDED.listed_on,
			programs     = EXCLUDED.programs,
			remarks      = EXCLUDED.remarks`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	h := sha256.New()
	for _, e := range entries {
		_, _ = h.Write([]byte(e.ExternalID + e.PrimaryName))
		_, err := stmt.ExecContext(ctx,
			string(source), e.ExternalID, e.EntityType, e.PrimaryName,
			pq.Array(e.Aliases), e.Nationality, e.BirthDate,
			e.ListedOn, pq.Array(e.Programs), e.Remarks,
		)
		if err != nil {
			return fmt.Errorf("sanctions: upsert entry %s: %w", e.ExternalID, err)
		}
	}

	checksum := fmt.Sprintf("%x", h.Sum(nil))[:16]
	_, err = tx.ExecContext(ctx, `
		INSERT INTO sanctions_lists (source, loaded_at, entry_count, checksum)
		VALUES ($1, NOW(), $2, $3)
		ON CONFLICT (source) DO UPDATE SET
			loaded_at   = NOW(),
			entry_count = $2,
			checksum    = $3`,
		string(source), len(entries), checksum,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Screen queries for entries whose primary_name is similar to subjectName
// using pg_trgm similarity. Returns matched entry IDs and scores.
func (r *Repository) Screen(ctx context.Context, subjectName string, threshold float64) ([]Entry, []float64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, list_source, external_id, entity_type, primary_name,
		       aliases, nationality, birth_date, listed_on, programs, remarks,
		       similarity(primary_name, $1) AS score
		FROM sanctions_entries
		WHERE similarity(primary_name, $1) >= $2
		   OR $1 ILIKE ANY(aliases)
		ORDER BY score DESC
		LIMIT 50`,
		subjectName, threshold,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var entries []Entry
	var scores []float64
	for rows.Next() {
		var e Entry
		var score float64
		var listedOn sql.NullTime
		if err := rows.Scan(
			&e.ID, &e.ListSource, &e.ExternalID, &e.EntityType, &e.PrimaryName,
			pq.Array(&e.Aliases), &e.Nationality, &e.BirthDate, &listedOn,
			pq.Array(&e.Programs), &e.Remarks, &score,
		); err != nil {
			return nil, nil, err
		}
		if listedOn.Valid {
			t := listedOn.Time
			e.ListedOn = &t
		}
		entries = append(entries, e)
		scores = append(scores, score)
	}
	return entries, scores, rows.Err()
}

// InsertHit records a screening match.
func (r *Repository) InsertHit(ctx context.Context, subjectID, entryID, matchType, matchedName string, score float64, screenedByUserID string) (string, error) {
	var id string
	var screenedBy interface{} = nil
	if screenedByUserID != "" {
		screenedBy = screenedByUserID
	}
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sanctions_hits
			(subject_id, entry_id, match_type, match_score, matched_name, screened_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text`,
		subjectID, entryID, matchType, score, matchedName, screenedBy,
	).Scan(&id)
	return id, err
}

// GetHitsBySubject returns all screening hits for a subject with their entry details.
func (r *Repository) GetHitsBySubject(ctx context.Context, subjectID string) ([]Hit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT h.id, h.subject_id::text, h.entry_id::text,
		       h.match_type, h.match_score, h.matched_name,
		       h.screened_at, h.screened_by::text, h.status,
		       h.reviewed_by::text, h.reviewed_at, h.notes,
		       e.list_source, e.external_id, e.entity_type, e.primary_name,
		       e.aliases, e.nationality, e.birth_date, e.listed_on, e.programs, e.remarks
		FROM sanctions_hits h
		JOIN sanctions_entries e ON e.id = h.entry_id
		WHERE h.subject_id = $1::uuid
		ORDER BY h.screened_at DESC`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

// GetAllOpenHits returns all unresolved hits across all subjects (for the settings page).
func (r *Repository) GetAllOpenHits(ctx context.Context, limit int) ([]Hit, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT h.id, h.subject_id::text, h.entry_id::text,
		       h.match_type, h.match_score, h.matched_name,
		       h.screened_at, h.screened_by::text, h.status,
		       h.reviewed_by::text, h.reviewed_at, h.notes,
		       e.list_source, e.external_id, e.entity_type, e.primary_name,
		       e.aliases, e.nationality, e.birth_date, e.listed_on, e.programs, e.remarks
		FROM sanctions_hits h
		JOIN sanctions_entries e ON e.id = h.entry_id
		WHERE h.status = 'open' OR h.status = 'confirmed' OR h.status = 'escalated'
		ORDER BY h.screened_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

// UpdateHitStatus updates the review status and notes on a hit.
func (r *Repository) UpdateHitStatus(ctx context.Context, hitID, status, notes, reviewerID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE sanctions_hits SET
			status      = $2,
			notes       = $3,
			reviewed_by = $4::uuid,
			reviewed_at = NOW()
		WHERE id = $1::uuid`,
		hitID, status, notes, reviewerID,
	)
	return err
}

// UpdateSubjectSanctionedFlag updates is_sanctioned + last_screened_at on the subject.
func (r *Repository) UpdateSubjectSanctionedFlag(ctx context.Context, subjectID string, isSanctioned bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE subjects SET is_sanctioned = $2, last_screened_at = NOW()
		WHERE id = $1::uuid`,
		subjectID, isSanctioned,
	)
	return err
}

// GetLists returns all sanctions list metadata rows.
func (r *Repository) GetLists(ctx context.Context) ([]SanctionsList, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, source, loaded_at, entry_count, checksum
		FROM sanctions_lists
		ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SanctionsList
	for rows.Next() {
		var l SanctionsList
		if err := rows.Scan(&l.ID, &l.Source, &l.LoadedAt, &l.EntryCount, &l.Checksum); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

func scanHits(rows *sql.Rows) ([]Hit, error) {
	var out []Hit
	for rows.Next() {
		var h Hit
		var screenedBy, reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		var e Entry
		var listedOn sql.NullTime
		if err := rows.Scan(
			&h.ID, &h.SubjectID, &h.EntryID,
			&h.MatchType, &h.MatchScore, &h.MatchedName,
			&h.ScreenedAt, &screenedBy, &h.Status,
			&reviewedBy, &reviewedAt, &h.Notes,
			&e.ListSource, &e.ExternalID, &e.EntityType, &e.PrimaryName,
			pq.Array(&e.Aliases), &e.Nationality, &e.BirthDate, &listedOn,
			pq.Array(&e.Programs), &e.Remarks,
		); err != nil {
			return nil, err
		}
		if screenedBy.Valid {
			h.ScreenedBy = &screenedBy.String
		}
		if reviewedBy.Valid {
			h.ReviewedBy = &reviewedBy.String
		}
		if reviewedAt.Valid {
			t := reviewedAt.Time
			h.ReviewedAt = &t
		}
		if listedOn.Valid {
			t := listedOn.Time
			e.ListedOn = &t
		}
		h.Entry = &e
		out = append(out, h)
	}
	return out, rows.Err()
}

// MarkListLoadedAt updates the loaded_at timestamp (used after a reload).
func (r *Repository) MarkListLoaded(ctx context.Context, source ListSource, count int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sanctions_lists (source, loaded_at, entry_count)
		VALUES ($1, $2, $3)
		ON CONFLICT (source) DO UPDATE SET loaded_at = $2, entry_count = $3`,
		string(source), time.Now().UTC(), count,
	)
	return err
}
