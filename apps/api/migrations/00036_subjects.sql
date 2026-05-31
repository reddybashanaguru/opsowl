-- +goose Up
-- +goose StatementBegin

CREATE TABLE subjects (
  id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id                UUID        NOT NULL,
  canonical_name        TEXT        NOT NULL,
  entity_type           TEXT        NOT NULL DEFAULT 'individual', -- individual, company, account, unknown
  -- Primary identity
  id_type               TEXT        NOT NULL DEFAULT '',
  id_number             TEXT        NOT NULL DEFAULT '',
  secondary_ids         JSONB       NOT NULL DEFAULT '[]', -- [{type,number,country,issued_by}]
  nationality           TEXT        NOT NULL DEFAULT '',
  date_of_birth         DATE,
  gender                TEXT        NOT NULL DEFAULT '',
  -- Contact
  addresses             JSONB       NOT NULL DEFAULT '[]', -- [{line1,city,state,country,zip,is_primary}]
  phones                TEXT[]      NOT NULL DEFAULT '{}',
  emails                TEXT[]      NOT NULL DEFAULT '{}',
  -- Risk profile
  risk_score            INTEGER     NOT NULL DEFAULT 0,         -- 0–100
  risk_level            TEXT        NOT NULL DEFAULT 'low',     -- low, medium, high, critical
  risk_factors          TEXT[]      NOT NULL DEFAULT '{}',
  is_pep                BOOLEAN     NOT NULL DEFAULT false,     -- Politically Exposed Person
  is_sanctioned         BOOLEAN     NOT NULL DEFAULT false,
  is_watchlisted        BOOLEAN     NOT NULL DEFAULT false,
  -- KYC
  kyc_tier              TEXT        NOT NULL DEFAULT '',        -- basic, standard, enhanced
  kyc_last_reviewed_at  TIMESTAMPTZ,
  -- Denormalised stats (updated on case link/unlink)
  total_cases           INTEGER     NOT NULL DEFAULT 0,
  open_cases            INTEGER     NOT NULL DEFAULT 0,
  total_str_filings     INTEGER     NOT NULL DEFAULT 0,
  -- Meta
  notes                 TEXT        NOT NULL DEFAULT '',
  created_by            UUID        REFERENCES users(id) ON DELETE SET NULL,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at            TIMESTAMPTZ,
  first_seen_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_seen_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_subjects_identity
  ON subjects (org_id, id_type, id_number)
  WHERE id_number != '' AND deleted_at IS NULL;

CREATE INDEX idx_subjects_org      ON subjects (org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_subjects_risk     ON subjects (org_id, risk_level, risk_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_subjects_pep      ON subjects (org_id, is_pep) WHERE is_pep = true AND deleted_at IS NULL;
CREATE INDEX idx_subjects_watchlist ON subjects (org_id, is_watchlisted) WHERE is_watchlisted = true AND deleted_at IS NULL;
CREATE INDEX idx_subjects_name     ON subjects USING gin (canonical_name gin_trgm_ops);

-- Case ↔ subject linking (many-to-many)
CREATE TABLE case_subjects (
  case_id    UUID        NOT NULL REFERENCES cases(id)    ON DELETE CASCADE,
  subject_id UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  role       TEXT        NOT NULL DEFAULT 'primary',  -- primary, secondary, counterparty, witness
  linked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  linked_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
  PRIMARY KEY (case_id, subject_id)
);

CREATE INDEX idx_case_subjects_subject ON case_subjects (subject_id);
CREATE INDEX idx_case_subjects_case    ON case_subjects (case_id);

-- Subject risk score audit trail
CREATE TABLE subject_risk_history (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  subject_id   UUID        NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
  risk_score   INTEGER     NOT NULL,
  risk_level   TEXT        NOT NULL,
  reason       TEXT        NOT NULL DEFAULT '',
  changed_by   TEXT        NOT NULL DEFAULT 'system', -- 'system' or user_id
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subject_risk_history ON subject_risk_history (subject_id, created_at DESC);

-- Enable trigram extension if not already enabled (needed for name search)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS subject_risk_history;
DROP TABLE IF EXISTS case_subjects;
DROP TABLE IF EXISTS subjects;
-- +goose StatementEnd
