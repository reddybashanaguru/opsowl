package documents

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

func NewRepository(q *db.Queries, sqlDB *sql.DB) *Repository {
	return &Repository{q: q, sqlDB: sqlDB}
}

// MarkStuckAsFailed finds documents that have been in 'processing' status for
// longer than staleAfter and marks them as 'failed'. Called at startup to
// recover from a crash-during-processing scenario.
func (r *Repository) MarkStuckAsFailed(ctx context.Context, staleAfter time.Duration) (int64, error) {
	cutoff := time.Now().Add(-staleAfter)
	res, err := r.sqlDB.ExecContext(ctx, `
		UPDATE documents
		SET processing_status = 'failed',
		    error_message = 'processing timeout: recovered at startup',
		    updated_at = NOW()
		WHERE processing_status = 'processing'
		  AND updated_at < $1
		  AND deleted_at IS NULL
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
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

func (r *Repository) ListAll(ctx context.Context, p db.ListAllDocumentsParams) ([]db.ListAllDocumentsRow, error) {
	return r.q.ListAllDocuments(ctx, p)
}

func (r *Repository) CountAll(ctx context.Context, status db.NullDocumentProcessingStatus) (int64, error) {
	return r.q.CountAllDocuments(ctx, status)
}

// FindBySHA256InCase returns the ID of an existing non-deleted document in the
// same case that has the given SHA-256 hex hash, or uuid.Nil if none found.
func (r *Repository) FindBySHA256InCase(ctx context.Context, caseID uuid.UUID, hash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.sqlDB.QueryRowContext(ctx, `
		SELECT id FROM documents
		WHERE case_id = $1
		  AND file_sha256 = $2
		  AND deleted_at IS NULL
		LIMIT 1
	`, caseID, hash).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	return id, err
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
