-- +goose Up
-- +goose StatementBegin

-- Idempotency key store: maps (key, user_id) → cached response.
-- Callers include Idempotency-Key: <uuid> on POST mutations; the middleware
-- replays the original response on retry rather than running the handler again.
-- This prevents double case creation, duplicate note saves, and double bulk-ops
-- under network retries or aggressive frontend retry logic.
--
-- TTL: 24 hours. Expired rows are cleaned up by a weekly scheduled DELETE.
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    idem_key        TEXT        NOT NULL,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method          TEXT        NOT NULL,
    path            TEXT        NOT NULL,
    response_status INT         NOT NULL DEFAULT 0,
    response_body   BYTEA,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);

-- Primary lookup: key + user guarantees per-user isolation
-- (user A's key never conflicts with user B's same key value)
CREATE UNIQUE INDEX IF NOT EXISTS idx_idem_key_user
    ON idempotency_keys (idem_key, user_id);

-- Cleanup index: find and delete expired rows efficiently
CREATE INDEX IF NOT EXISTS idx_idem_expires
    ON idempotency_keys (expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idempotency_keys;
-- +goose StatementEnd
