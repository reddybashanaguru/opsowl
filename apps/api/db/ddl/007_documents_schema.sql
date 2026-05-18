-- Documents domain: uploads, processing state, pgvector chunks for RAG.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE documents (
    id                UUID                        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id           UUID                        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    uploaded_by       UUID                        NOT NULL REFERENCES users(id),
    original_filename TEXT                        NOT NULL,
    stored_path       TEXT                        NOT NULL,
    file_type         TEXT                        NOT NULL,
    mime_type         TEXT                        NOT NULL,
    file_size_bytes   BIGINT                      NOT NULL,
    file_sha256       TEXT,                       -- SHA-256 hex for chain-of-custody verification
    processing_status document_processing_status  NOT NULL DEFAULT 'pending',
    page_count        INTEGER,
    chunk_count       INTEGER,
    error_message     TEXT,
    last_error_code   TEXT,
    retry_count       INT                         NOT NULL DEFAULT 0,
    doc_category      TEXT,                       -- AI-assigned: contract/invoice/identity/evidence/other
    search_vector     TSVECTOR,
    created_at        TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);
CREATE INDEX idx_documents_case_id           ON documents (case_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_uploaded_by       ON documents (uploaded_by);
CREATE INDEX idx_documents_processing_status ON documents (processing_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_search            ON documents USING GIN (search_vector);

-- pgvector chunks for RAG pipeline. HNSW index for sub-millisecond ANN search.
CREATE TABLE document_chunks (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID         NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    case_id     UUID         NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    chunk_index INTEGER      NOT NULL,
    content     TEXT         NOT NULL,
    embedding   vector(1536) NOT NULL,           -- OpenAI ada-002 / nomic-embed-text (Ollama)
    page_number INTEGER,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chunks_document_id ON document_chunks (document_id);
CREATE INDEX idx_chunks_case_id     ON document_chunks (case_id);
CREATE INDEX idx_chunks_embedding   ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
