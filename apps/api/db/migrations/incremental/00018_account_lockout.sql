-- +goose Up
-- +goose StatementBegin

-- Account lockout: tracks consecutive failed login attempts per user.
-- After max_attempts (default 5) failures the account is locked for lockout_minutes.
-- Lockout duration doubles with each consecutive lockout event (progressive lockout),
-- capped at 24 hours — aligns with NIST SP 800-63B and RBI CSF account security controls.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS failed_login_attempts  INT         NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS locked_until           TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS lockout_count          INT         NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users (locked_until) WHERE locked_until IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
  DROP COLUMN IF EXISTS failed_login_attempts,
  DROP COLUMN IF EXISTS locked_until,
  DROP COLUMN IF EXISTS lockout_count;
-- +goose StatementEnd
