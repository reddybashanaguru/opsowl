-- +goose Up
-- +goose StatementBegin

-- Registered TMS / alert-feed sources
CREATE TABLE alert_sources (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id        UUID        NOT NULL,
  name          TEXT        NOT NULL,
  source_type   TEXT        NOT NULL DEFAULT 'webhook',  -- webhook, sftp, manual, csv
  source_system TEXT        NOT NULL DEFAULT 'generic',  -- actimize, sas_aml, oracle_fccm, featurespace, generic
  webhook_secret TEXT       NOT NULL DEFAULT '',         -- HMAC-SHA256 signing secret
  description   TEXT        NOT NULL DEFAULT '',
  is_active     BOOLEAN     NOT NULL DEFAULT true,
  created_by    UUID        REFERENCES users(id) ON DELETE SET NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_sources_org ON alert_sources (org_id) WHERE is_active = true;

-- Routing rules: alert_type + risk_score → priority, assignment, playbook
CREATE TABLE alert_rules (
  id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id               UUID        NOT NULL,
  name                 TEXT        NOT NULL,
  alert_type           TEXT        NOT NULL DEFAULT '*',     -- specific type or '*' catch-all
  min_risk_score       INTEGER     NOT NULL DEFAULT 0,       -- minimum risk_score to match
  priority_mapping     TEXT        NOT NULL DEFAULT 'medium',-- critical | high | medium | low
  tags                 TEXT[]      NOT NULL DEFAULT '{}',
  auto_assign_user_id  UUID        REFERENCES users(id) ON DELETE SET NULL,
  playbook_case_type   TEXT        NOT NULL DEFAULT '',      -- auto-apply playbook by case_type
  dedup_window_hours   INTEGER     NOT NULL DEFAULT 72,      -- 0 = no dedup
  mode                 TEXT        NOT NULL DEFAULT 'queue', -- queue | auto
  is_active            BOOLEAN     NOT NULL DEFAULT true,
  sort_order           INTEGER     NOT NULL DEFAULT 0,       -- lower = higher priority
  created_by           UUID        REFERENCES users(id) ON DELETE SET NULL,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_org ON alert_rules (org_id, sort_order) WHERE is_active = true;

-- Ingested alerts (normalised canonical model)
CREATE TABLE incoming_alerts (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id              UUID        NOT NULL,
  source_id           UUID        REFERENCES alert_sources(id) ON DELETE SET NULL,
  external_alert_id   TEXT        NOT NULL DEFAULT '',   -- ID in the originating TMS
  alert_type          TEXT        NOT NULL,
  risk_score          INTEGER     NOT NULL DEFAULT 0,    -- 0–100
  risk_level          TEXT        NOT NULL DEFAULT 'medium', -- low | medium | high | critical
  subject_name        TEXT        NOT NULL DEFAULT '',
  subject_id_type     TEXT        NOT NULL DEFAULT '',
  subject_id_number   TEXT        NOT NULL DEFAULT '',
  account_number      TEXT        NOT NULL DEFAULT '',
  amount              NUMERIC     NOT NULL DEFAULT 0,
  currency            TEXT        NOT NULL DEFAULT '',
  transaction_date    TIMESTAMPTZ,
  description         TEXT        NOT NULL DEFAULT '',
  source_system       TEXT        NOT NULL DEFAULT 'generic',
  raw_payload         JSONB       NOT NULL DEFAULT '{}', -- original payload preserved
  -- Workflow
  status              TEXT        NOT NULL DEFAULT 'pending', -- pending | reviewing | converted | dismissed | duplicate
  linked_case_id      UUID        REFERENCES cases(id)    ON DELETE SET NULL,
  matched_rule_id     UUID        REFERENCES alert_rules(id) ON DELETE SET NULL,
  subject_id          UUID        REFERENCES subjects(id) ON DELETE SET NULL,
  -- Actions
  dismissed_by        UUID        REFERENCES users(id) ON DELETE SET NULL,
  dismissed_at        TIMESTAMPTZ,
  dismiss_reason      TEXT        NOT NULL DEFAULT '',
  reviewed_by         UUID        REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at         TIMESTAMPTZ,
  converted_by        UUID        REFERENCES users(id) ON DELETE SET NULL,
  converted_at        TIMESTAMPTZ,
  -- Meta
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incoming_alerts_org_status ON incoming_alerts (org_id, status, created_at DESC);
CREATE INDEX idx_incoming_alerts_type       ON incoming_alerts (org_id, alert_type);
CREATE INDEX idx_incoming_alerts_subject    ON incoming_alerts (subject_id) WHERE subject_id IS NOT NULL;
CREATE INDEX idx_incoming_alerts_case       ON incoming_alerts (linked_case_id) WHERE linked_case_id IS NOT NULL;
CREATE UNIQUE INDEX idx_incoming_alerts_dedup
  ON incoming_alerts (org_id, source_id, external_alert_id)
  WHERE external_alert_id != '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS incoming_alerts;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS alert_sources;
-- +goose StatementEnd
