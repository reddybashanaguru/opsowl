-- +goose Up
-- +goose StatementBegin
-- Switch embedding dimension from 1536 (OpenAI text-embedding-3-small) to
-- 768 (Ollama nomic-embed-text). The default on-prem deployment uses Ollama.
-- Existing chunks are deleted — they were generated with OpenAI 1536-dim vectors
-- and are incompatible with the new dimension. Re-upload documents to re-index.
-- If purchasing the Cloud AI add-on (OpenAI), run the inverse migration first.
DROP INDEX IF EXISTS idx_chunks_embedding;
DELETE FROM document_chunks;
ALTER TABLE document_chunks ALTER COLUMN embedding TYPE vector(768);
CREATE INDEX idx_chunks_embedding ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_chunks_embedding;
DELETE FROM document_chunks;
ALTER TABLE document_chunks ALTER COLUMN embedding TYPE vector(1536);
CREATE INDEX idx_chunks_embedding ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
-- +goose StatementEnd
