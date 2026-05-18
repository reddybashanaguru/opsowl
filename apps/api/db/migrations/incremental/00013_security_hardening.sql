-- +goose Up

-- ── 1. Document SHA-256 integrity hash ───────────────────────────────────────
ALTER TABLE documents ADD COLUMN IF NOT EXISTS file_sha256 TEXT;

-- ── 2. Immutable audit_logs — trigger blocks UPDATE/DELETE for all roles ─────
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only. Modifications are not permitted (event_code: %)', OLD.event_code;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER audit_logs_immutable
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

-- ── 3. SSE short-lived ticket table ──────────────────────────────────────────
CREATE TABLE sse_tickets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sse_tickets_hash    ON sse_tickets (token_hash);
CREATE INDEX idx_sse_tickets_user    ON sse_tickets (user_id);
CREATE INDEX idx_sse_tickets_expires ON sse_tickets (expires_at);

-- +goose Down

DROP INDEX IF EXISTS idx_sse_tickets_expires;
DROP INDEX IF EXISTS idx_sse_tickets_user;
DROP INDEX IF EXISTS idx_sse_tickets_hash;
DROP TABLE IF EXISTS sse_tickets;

DROP TRIGGER IF EXISTS audit_logs_immutable ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_modification();

ALTER TABLE documents DROP COLUMN IF EXISTS file_sha256;
