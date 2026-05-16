-- +goose Up
-- +goose StatementBegin

CREATE TYPE user_role AS ENUM (
  'super_admin', 'admin', 'manager', 'analyst', 'viewer'
);

CREATE TABLE users (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email          TEXT        NOT NULL UNIQUE,
  password_hash  TEXT        NOT NULL,
  full_name      TEXT        NOT NULL,
  role           user_role   NOT NULL DEFAULT 'analyst',
  is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
  last_login_at  TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_users_email      ON users (email)     WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role       ON users (role)      WHERE deleted_at IS NULL;
CREATE INDEX idx_users_is_active  ON users (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users (created_at);

CREATE TABLE refresh_tokens (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  TEXT        NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked_at  TIMESTAMPTZ,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rt_user_id    ON refresh_tokens (user_id);
CREATE INDEX idx_rt_expires_at ON refresh_tokens (expires_at);
CREATE INDEX idx_rt_hash       ON refresh_tokens (token_hash) WHERE revoked_at IS NULL;

CREATE TABLE sessions (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  ip_address     INET        NOT NULL,
  user_agent     TEXT        NOT NULL DEFAULT '',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id     ON sessions (user_id);
CREATE INDEX idx_sessions_last_active ON sessions (last_active_at);

CREATE TABLE audit_logs (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  event_code    TEXT        NOT NULL,
  actor_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
  ip_address    TEXT,
  resource_type TEXT,
  resource_id   UUID,
  metadata      JSONB       NOT NULL DEFAULT '{}',
  severity      TEXT        NOT NULL DEFAULT 'info',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_event_code  ON audit_logs (event_code);
CREATE INDEX idx_audit_actor_id    ON audit_logs (actor_id);
CREATE INDEX idx_audit_created_at  ON audit_logs (created_at);
CREATE INDEX idx_audit_resource    ON audit_logs (resource_type, resource_id) WHERE resource_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
DROP TYPE  IF EXISTS user_role;
-- +goose StatementEnd
