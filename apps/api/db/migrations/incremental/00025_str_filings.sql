-- +goose Up
-- +goose StatementBegin

-- STR/SAR/CTR filing workflow with PMLA-mandated dual-authorization (maker-checker).
-- PMLA 2002 s.12: every STR must be reviewed and approved by a person other
-- than the person who prepared it. Enforced at both DB and application layer.

CREATE TYPE str_status AS ENUM ('draft', 'submitted', 'approved', 'rejected', 'filed');

CREATE TABLE IF NOT EXISTS str_filings (
  id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id          UUID         NOT NULL REFERENCES cases(id) ON DELETE RESTRICT,
  org_id           UUID         NOT NULL,
  report_type      TEXT         NOT NULL DEFAULT 'STR',  -- STR | SAR | CTR
  status           str_status   NOT NULL DEFAULT 'draft',

  -- Maker-Checker identities
  submitted_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
  approved_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
  submitted_at     TIMESTAMPTZ,
  approved_at      TIMESTAMPTZ,
  filed_at         TIMESTAMPTZ,
  rejection_reason TEXT,
  fiu_reference    TEXT,         -- FIU-IND acknowledgement ref after filing

  -- Report payload (stored as JSONB for schema flexibility)
  subject          JSONB        NOT NULL DEFAULT '{}',
  transactions     JSONB        NOT NULL DEFAULT '[]',
  filing_entity    JSONB        NOT NULL DEFAULT '{}',
  narrative        TEXT         NOT NULL DEFAULT '',
  suspicion_basis  TEXT         NOT NULL,
  evidence_summary TEXT         NOT NULL DEFAULT '',

  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

  -- DB-layer maker-checker guard: submitter and approver must be different people.
  -- NULL values are allowed (draft has neither; submitted has only submitted_by).
  CONSTRAINT chk_str_maker_checker
    CHECK (submitted_by IS NULL OR approved_by IS NULL OR submitted_by <> approved_by)
);

CREATE INDEX IF NOT EXISTS idx_str_filings_case   ON str_filings (case_id);
CREATE INDEX IF NOT EXISTS idx_str_filings_org    ON str_filings (org_id, status);
CREATE INDEX IF NOT EXISTS idx_str_filings_status ON str_filings (status) WHERE filed_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS str_filings;
DROP TYPE IF EXISTS str_status;
-- +goose StatementEnd
