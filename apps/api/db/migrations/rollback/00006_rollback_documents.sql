
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
