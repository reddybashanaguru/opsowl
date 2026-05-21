-- +goose Up
-- +goose StatementBegin

-- Add device fingerprint and last-active tracking so the session limit
-- enforcement can revoke the oldest session when the per-user cap is reached.
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS device_name  TEXT,
  ADD COLUMN IF NOT EXISTS is_active    BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS revoked_at   TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_sessions_active_user ON sessions (user_id, last_active_at DESC)
  WHERE is_active = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions
  DROP COLUMN IF EXISTS device_name,
  DROP COLUMN IF EXISTS is_active,
  DROP COLUMN IF EXISTS revoked_at;
-- +goose StatementEnd
