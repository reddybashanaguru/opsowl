package ai

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
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
