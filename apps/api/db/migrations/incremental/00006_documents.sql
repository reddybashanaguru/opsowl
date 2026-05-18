-- +goose Up

CREATE TYPE document_processing_status AS ENUM (
    'pending',
    'processing',
    'completed',
    'failed'
);

CREATE TABLE documents (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id               UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    uploaded_by           UUID        NOT NULL REFERENCES users(id),
    original_filename     TEXT        NOT NULL,
    stored_path           TEXT        NOT NULL,
    file_type             TEXT        NOT NULL,
    mime_type             TEXT        NOT NULL,
    file_size_bytes       BIGINT      NOT NULL,
    processing_status     document_processing_status NOT NULL DEFAULT 'pending',
    page_count            INTEGER,
    chunk_count           INTEGER,
    error_message         TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ
);

CREATE INDEX idx_documents_case_id            ON documents (case_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_uploaded_by        ON documents (uploaded_by);
CREATE INDEX idx_documents_processing_status  ON documents (processing_status) WHERE deleted_at IS NULL;

CREATE TRIGGER documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

CREATE TABLE document_chunks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID        NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    case_id      UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    chunk_index  INTEGER     NOT NULL,
    content      TEXT        NOT NULL,
    embedding    vector(1536) NOT NULL,
    page_number  INTEGER,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_document_id ON document_chunks (document_id);
CREATE INDEX idx_chunks_case_id     ON document_chunks (case_id);
CREATE INDEX idx_chunks_embedding   ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- +goose Down

DROP INDEX IF EXISTS idx_chunks_embedding;
DROP INDEX IF EXISTS idx_chunks_case_id;
DROP INDEX IF EXISTS idx_chunks_document_id;
DROP TABLE IF EXISTS document_chunks;

DROP TRIGGER IF EXISTS documents_updated_at ON documents;
DROP INDEX IF EXISTS idx_documents_processing_status;
DROP INDEX IF EXISTS idx_documents_uploaded_by;
DROP INDEX IF EXISTS idx_documents_case_id;
DROP TABLE IF EXISTS documents;

DROP TYPE IF EXISTS document_processing_status;
