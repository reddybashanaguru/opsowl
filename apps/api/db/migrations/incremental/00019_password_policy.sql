-- +goose Up
-- +goose StatementBegin

-- Password history: stores hashed previous passwords to prevent reuse.
-- The application enforces a configurable history count (default: last 5 passwords).
CREATE TABLE IF NOT EXISTS password_history (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  password_hash TEXT       NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pwd_history_user ON password_history (user_id, created_at DESC);

-- Track when the password was last changed and when it expires.
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS password_expires_at TIMESTAMPTZ;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS password_history;
ALTER TABLE users
  DROP COLUMN IF EXISTS password_changed_at,
  DROP COLUMN IF EXISTS password_expires_at;
-- +goose StatementEnd
