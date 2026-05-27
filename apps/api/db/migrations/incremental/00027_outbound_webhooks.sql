-- +goose Up
-- +goose StatementBegin

CREATE TABLE outbound_webhooks (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name            TEXT         NOT NULL,
  url             TEXT         NOT NULL,
  signing_secret  TEXT         NOT NULL,
  event_types     TEXT[]       NOT NULL DEFAULT '{}',
  is_active       BOOLEAN      NOT NULL DEFAULT true,
  created_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);

CREATE TABLE outbound_webhook_deliveries (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  webhook_id      UUID         NOT NULL REFERENCES outbound_webhooks(id) ON DELETE CASCADE,
  event_type      TEXT         NOT NULL,
  case_id         UUID         REFERENCES cases(id) ON DELETE SET NULL,
  payload         JSONB        NOT NULL,
  status          TEXT         NOT NULL DEFAULT 'pending',
  attempt_count   INT          NOT NULL DEFAULT 0,
  next_attempt_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  last_status     INT,
  last_error      TEXT,
  delivered_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_pending
  ON outbound_webhook_deliveries(next_attempt_at)
  WHERE status = 'pending';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_webhook_deliveries_pending;
DROP TABLE IF EXISTS outbound_webhook_deliveries;
DROP TABLE IF EXISTS outbound_webhooks;
-- +goose StatementEnd
