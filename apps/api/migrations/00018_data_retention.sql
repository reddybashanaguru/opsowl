-- +goose Up
-- +goose StatementBegin

-- Data retention policies aligned with DPDP Act 2023 and RBI record-keeping requirements.
-- The purge_job background service reads this table and deletes records older than retention_days.
CREATE TABLE IF NOT EXISTS data_retention_policies (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  resource_type   TEXT        NOT NULL UNIQUE,  -- 'audit_logs', 'notifications', 'sessions', etc.
  retention_days  INT         NOT NULL,
  purge_enabled   BOOLEAN     NOT NULL DEFAULT TRUE,
  last_purged_at  TIMESTAMPTZ,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by      UUID        REFERENCES users(id) ON DELETE SET NULL
);

-- Default retention periods (regulatory minimums for Indian financial services)
INSERT INTO data_retention_policies (resource_type, retention_days, purge_enabled) VALUES
  ('audit_logs',     730,  FALSE),   -- 2 years; RBI requires audit log retention; purge_enabled=FALSE = keep forever unless explicitly enabled
  ('cases',          2555, FALSE),   -- 7 years; regulatory record-keeping requirement
  ('documents',      2555, FALSE),   -- 7 years; match case retention
  ('notifications',  90,   TRUE),    -- 90 days; no regulatory requirement
  ('sessions',       30,   TRUE),    -- 30 days; short-lived operational data
  ('ai_logs',        365,  FALSE)    -- 1 year; AI audit trail
ON CONFLICT (resource_type) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS data_retention_policies;
-- +goose StatementEnd
