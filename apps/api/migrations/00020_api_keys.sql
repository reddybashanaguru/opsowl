-- +goose Up
-- +goose StatementBegin

-- API keys for machine-to-machine integrations (Finacle, CBS, third-party alert feeds).
-- The full key is shown ONCE on creation; only the SHA-256 hash is stored.
-- key_prefix (first 8 chars) allows the user to identify the key without exposing it.
CREATE TABLE IF NOT EXISTS api_keys (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name          TEXT        NOT NULL,
  key_hash      TEXT        NOT NULL UNIQUE,   -- SHA-256(key)
  key_prefix    TEXT        NOT NULL,           -- first 8 chars for display
  scopes        TEXT[]      NOT NULL DEFAULT '{}',  -- e.g. 'cases:read', 'cases:write', 'inbound:write'
  last_used_at  TIMESTAMPTZ,
  expires_at    TIMESTAMPTZ,
  revoked_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user   ON api_keys (user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_hash   ON api_keys (key_hash) WHERE revoked_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS api_keys;
-- +goose StatementEnd
