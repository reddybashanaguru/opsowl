package documents

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

func (r *Repository) Create(ctx context.Context, p db.CreateDocumentParams) (db.Document, error) {
	return r.q.CreateDocument(ctx, p)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (db.GetDocumentByIDRow, error) {
	row, err := r.q.GetDocumentByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.GetDocumentByIDRow{}, ErrNotFound
		}
		return db.GetDocumentByIDRow{}, err
	}
	return row, nil
}

func (r *Repository) ListByCase(ctx context.Context, p db.ListDocumentsByCaseParams) ([]db.ListDocumentsByCaseRow, error) {
	return r.q.ListDocumentsByCase(ctx, p)
}

func (r *Repository) Count(ctx context.Context, caseID uuid.UUID) (int64, error) {
	return r.q.CountDocumentsByCase(ctx, caseID)
}

func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status db.DocumentProcessingStatus) (db.Document, error) {
	return r.q.UpdateDocumentStatus(ctx, db.UpdateDocumentStatusParams{
		ID:               id,
		ProcessingStatus: status,
	})
}

func (r *Repository) UpdateProcessingResult(ctx context.Context, p db.UpdateDocumentProcessingResultParams) (db.Document, error) {
	return r.q.UpdateDocumentProcessingResult(ctx, p)
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.q.SoftDeleteDocument(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
