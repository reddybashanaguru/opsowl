package ai

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q     *db.Queries
	sqlDB *sql.DB
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

// WithDB attaches a raw *sql.DB for queries that need joins not in sqlc schema.
func (r *Repository) WithDB(d *sql.DB) *Repository {
	r.sqlDB = d
	return r
}

// AIExportRow is a flat row for EU AI Act audit CSV export.
type AIExportRow struct {
	ID                uuid.UUID
	CaseID            uuid.UUID
	CaseNumber        string
	ActionType        string
	ModelUsed         string
	Provider          string
	ConfidenceScore   *float64
	HumanReviewStatus string
	ReviewedBy        *uuid.UUID
	ReviewedAt        *time.Time
	CreatedAt         time.Time
}

// ListForExport returns all AI log rows in a date window joined with case numbers.
func (r *Repository) ListForExport(ctx context.Context, from, to time.Time) ([]AIExportRow, error) {
	if r.sqlDB == nil {
		return nil, errors.New("ai: raw DB not attached; call WithDB before ListForExport")
	}
	rows, err := r.sqlDB.QueryContext(ctx, `
		SELECT l.id, l.case_id, c.case_number,
		       l.action_type, l.model_used, l.provider,
		       l.confidence_score,
		       l.human_review_status,
		       l.reviewed_by, l.reviewed_at,
		       l.created_at
		FROM ai_logs l
		LEFT JOIN cases c ON c.id = l.case_id
		WHERE l.created_at >= $1 AND l.created_at < $2
		ORDER BY l.created_at DESC
		LIMIT 10000`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AIExportRow
	for rows.Next() {
		var row AIExportRow
		var conf sql.NullFloat64
		var reviewedBy uuid.NullUUID
		var reviewedAt sql.NullTime
		if err := rows.Scan(
			&row.ID, &row.CaseID, &row.CaseNumber,
			&row.ActionType, &row.ModelUsed, &row.Provider,
			&conf, &row.HumanReviewStatus,
			&reviewedBy, &reviewedAt,
			&row.CreatedAt,
		); err != nil {
			return nil, err
		}
		if conf.Valid {
			row.ConfidenceScore = &conf.Float64
		}
		if reviewedBy.Valid {
			id := reviewedBy.UUID
			row.ReviewedBy = &id
		}
		if reviewedAt.Valid {
			t := reviewedAt.Time
			row.ReviewedAt = &t
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) Create(ctx context.Context, arg db.CreateAILogParams) (db.GetAILogRow, error) {
	log, err := r.q.CreateAILog(ctx, arg)
	if err != nil {
		return db.GetAILogRow{}, err
	}
	// Fetch again so we get the joined user name fields.
	return r.q.GetAILog(ctx, log.ID)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.GetAILogRow, error) {
	row, err := r.q.GetAILog(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.GetAILogRow{}, ErrNotFound
	}
	return row, err
}

func (r *Repository) ListByCase(ctx context.Context, arg db.ListAILogsByCaseParams) ([]db.ListAILogsByCaseRow, error) {
	return r.q.ListAILogsByCase(ctx, arg)
}

func (r *Repository) CountByCase(ctx context.Context, caseID uuid.UUID, status string) (int64, error) {
	return r.q.CountAILogsByCase(ctx, db.CountAILogsByCaseParams{
		CaseID:  caseID,
		Column2: status,
	})
}

func (r *Repository) ListPending(ctx context.Context, limit int32) ([]db.ListPendingAILogsRow, error) {
	return r.q.ListPendingAILogs(ctx, limit)
}

func (r *Repository) Approve(ctx context.Context, id, reviewerID uuid.UUID) (db.AiLog, error) {
	log, err := r.q.ApproveAILog(ctx, db.ApproveAILogParams{
		ID:         id,
		ReviewedBy: uuid.NullUUID{UUID: reviewerID, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.AiLog{}, ErrNotFound
	}
	return log, err
}

func (r *Repository) Reject(ctx context.Context, id, reviewerID uuid.UUID, reason string) (db.AiLog, error) {
	log, err := r.q.RejectAILog(ctx, db.RejectAILogParams{
		ID:                  id,
		ReviewedBy:          uuid.NullUUID{UUID: reviewerID, Valid: true},
		HumanEditedResponse: sql.NullString{String: reason, Valid: reason != ""},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.AiLog{}, ErrNotFound
	}
	return log, err
}

func (r *Repository) Edit(ctx context.Context, id, reviewerID uuid.UUID, editedResponse string) (db.AiLog, error) {
	log, err := r.q.EditAILog(ctx, db.EditAILogParams{
		ID:                  id,
		ReviewedBy:          uuid.NullUUID{UUID: reviewerID, Valid: true},
		HumanEditedResponse: sql.NullString{String: editedResponse, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return db.AiLog{}, ErrNotFound
	}
	return log, err
}
