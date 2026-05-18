-- +goose Up
-- +goose StatementBegin

CREATE TYPE case_status AS ENUM (
  'draft', 'open', 'in_progress', 'pending_review', 'closed', 'escalated'
);

CREATE TYPE case_priority AS ENUM ('critical', 'high', 'medium', 'low');

CREATE TABLE cases (
  id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
  case_number     TEXT          NOT NULL UNIQUE,
  title           TEXT          NOT NULL CHECK(length(title) >= 5),
  description     TEXT          NOT NULL DEFAULT '',
  status          case_status   NOT NULL DEFAULT 'draft',
  priority        case_priority NOT NULL DEFAULT 'medium',
  assigned_to     UUID          REFERENCES users(id),
  created_by      UUID          NOT NULL REFERENCES users(id),
  organization_id UUID          NOT NULL,
  due_date        TIMESTAMPTZ,
  sla_breach_at   TIMESTAMPTZ,
  sla_breached    BOOLEAN       NOT NULL DEFAULT FALSE,
  tags            TEXT[]        NOT NULL DEFAULT '{}',
  created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_cases_status         ON cases (status)          WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_priority        ON cases (priority)        WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_assigned_to     ON cases (assigned_to)     WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_created_by      ON cases (created_by);
CREATE INDEX idx_cases_organization_id ON cases (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_sla_breach_at   ON cases (sla_breach_at)   WHERE deleted_at IS NULL AND sla_breached = FALSE;
CREATE INDEX idx_cases_created_at      ON cases (created_at DESC);
CREATE INDEX idx_cases_tags            ON cases USING GIN (tags);

CREATE SEQUENCE case_number_seq START 1 INCREMENT 1 NO CYCLE;

CREATE TABLE case_assignments (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id       UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  assigned_to   UUID        NOT NULL REFERENCES users(id),
  assigned_by   UUID        NOT NULL REFERENCES users(id),
  assigned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unassigned_at TIMESTAMPTZ
);

CREATE INDEX idx_ca_case_id     ON case_assignments (case_id);
CREATE INDEX idx_ca_assigned_to ON case_assignments (assigned_to);

CREATE TABLE case_tags (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT        NOT NULL UNIQUE,
  color_hex  CHAR(7)     NOT NULL DEFAULT '#64748b',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE case_watchers (
  case_id    UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  user_id    UUID        NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (case_id, user_id)
);

CREATE INDEX idx_cw_user_id ON case_watchers (user_id);

CREATE TABLE case_activities (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id    UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  actor_id   UUID        NOT NULL REFERENCES users(id),
  event_type TEXT        NOT NULL,
  payload    JSONB       NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cact_case_id    ON case_activities (case_id);
CREATE INDEX idx_cact_created_at ON case_activities (case_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS case_activities;
DROP TABLE IF EXISTS case_watchers;
DROP TABLE IF EXISTS case_tags;
DROP TABLE IF EXISTS case_assignments;
DROP TABLE IF EXISTS cases;
DROP SEQUENCE IF EXISTS case_number_seq;
DROP TYPE IF EXISTS case_priority;
DROP TYPE IF EXISTS case_status;
-- +goose StatementEnd
