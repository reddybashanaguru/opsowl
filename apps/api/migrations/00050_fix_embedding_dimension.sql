-- +goose Up
-- Fix embedding vector dimension: nomic-embed-text outputs 768-dim vectors,
-- but the schema was created for OpenAI text-embedding-3-small (1536-dim).
-- All existing chunk data is invalid (wrong dimension) so we truncate and rebuild.

DROP INDEX IF EXISTS idx_chunks_embedding;
DELETE FROM document_chunks;
ALTER TABLE document_chunks ALTER COLUMN embedding TYPE vector(768);

CREATE INDEX idx_chunks_embedding ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- +goose Down
DROP INDEX IF EXISTS idx_chunks_embedding;
DELETE FROM document_chunks;
ALTER TABLE document_chunks ALTER COLUMN embedding TYPE vector(1536);

CREATE INDEX idx_chunks_embedding ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
