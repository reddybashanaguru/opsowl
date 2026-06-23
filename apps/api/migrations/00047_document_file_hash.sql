-- +goose Up

-- SHA-256 hex digest for chain-of-custody verification and duplicate detection.
ALTER TABLE documents ADD COLUMN IF NOT EXISTS file_sha256 TEXT;

CREATE INDEX IF NOT EXISTS idx_documents_sha256 ON documents (case_id, file_sha256)
    WHERE deleted_at IS NULL AND file_sha256 IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_documents_sha256;
ALTER TABLE documents DROP COLUMN IF EXISTS file_sha256;
