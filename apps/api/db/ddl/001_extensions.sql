-- PostgreSQL extensions required by OpsOwl.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- gen_random_uuid() fallback
CREATE EXTENSION IF NOT EXISTS vector;        -- pgvector for AI embeddings
CREATE EXTENSION IF NOT EXISTS pg_trgm;       -- trigram similarity for full-text search
