-- Extensions and custom types/enums for OpsOwl
-- This is a reference file. Do NOT run directly — use migrations/incremental/.

-- ── Extensions ────────────────────────────────────────────────────────────────
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ── Enums ─────────────────────────────────────────────────────────────────────
CREATE TYPE user_role AS ENUM (
    'viewer', 'analyst', 'manager', 'admin', 'super_admin'
);

CREATE TYPE case_status AS ENUM (
    'draft', 'open', 'in_progress', 'pending_review',
    'escalated', 'closed'
);

CREATE TYPE case_priority AS ENUM (
    'low', 'medium', 'high', 'critical'
);

CREATE TYPE doc_processing_status AS ENUM (
    'pending', 'processing', 'completed', 'failed'
);

CREATE TYPE timeline_event_source AS ENUM (
    'system', 'user', 'ai'
);
