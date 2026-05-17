-- name: CreateDocument :one
INSERT INTO documents (id, case_id, uploaded_by, original_filename, stored_path, file_type, mime_type, file_size_bytes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetDocumentByID :one
SELECT
    d.*,
    u.full_name  AS uploaded_by_name,
    u.email      AS uploaded_by_email
FROM documents d
JOIN users u ON u.id = d.uploaded_by
WHERE d.id = $1 AND d.deleted_at IS NULL;

-- name: ListDocumentsByCase :many
SELECT
    d.*,
    u.full_name  AS uploaded_by_name,
    u.email      AS uploaded_by_email
FROM documents d
JOIN users u ON u.id = d.uploaded_by
WHERE d.case_id = $1
  AND d.deleted_at IS NULL
  AND (sqlc.narg('status')::document_processing_status IS NULL OR d.processing_status = sqlc.narg('status'))
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR d.id < sqlc.narg('cursor_id'))
ORDER BY d.created_at DESC, d.id DESC
LIMIT @row_limit;

-- name: CountDocumentsByCase :one
SELECT COUNT(*) FROM documents
WHERE case_id = $1 AND deleted_at IS NULL;

-- name: UpdateDocumentStatus :one
UPDATE documents SET processing_status = $2
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateDocumentProcessingResult :one
UPDATE documents SET
    processing_status = $2,
    page_count        = sqlc.narg('page_count'),
    chunk_count       = sqlc.narg('chunk_count'),
    error_message     = sqlc.narg('error_message')
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDocument :one
UPDATE documents SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;
