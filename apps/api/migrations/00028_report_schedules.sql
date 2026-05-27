-- +goose Up
-- +goose StatementBegin

-- Scheduled case summary email reports.
-- frequency: 'daily' | 'weekly'
-- filters: {"status":"open","priority":"high","sla_breached":true} — any field optional
-- day_of_week: 0=Sun … 6=Sat, used only for weekly
CREATE TABLE report_schedules (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name          TEXT         NOT NULL,
  frequency     TEXT         NOT NULL DEFAULT 'daily',
  recipients    TEXT[]       NOT NULL DEFAULT '{}',
  filters       JSONB        NOT NULL DEFAULT '{}',
  send_hour     INT          NOT NULL DEFAULT 7,
  day_of_week   INT,
  is_active     BOOLEAN      NOT NULL DEFAULT true,
  last_sent_at  TIMESTAMPTZ,
  next_run_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  created_by    UUID         REFERENCES users(id) ON DELETE SET NULL,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_schedules_due ON report_schedules(next_run_at) WHERE is_active = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_report_schedules_due;
DROP TABLE IF EXISTS report_schedules;
-- +goose StatementEnd
