-- +goose Up
-- +goose StatementBegin

-- ── Closure disposition codes ──────────────────────────────────────────────
-- These are the regulatory disposition options an investigator must choose
-- when closing a case. Required by PMLA / FIU-IND for audit defensibility.
CREATE TYPE closure_disposition AS ENUM (
  'sar_filed',                 -- STR submitted to FIU-IND
  'no_sar_warranted',          -- investigated, no reportable suspicious activity
  'false_positive',            -- alert was incorrect, no suspicious activity
  'closed_monitoring',         -- closed but subject placed on enhanced watchlist
  'escalated_law_enforcement', -- referred to ED / CBI / state police
  'referred_internal'          -- passed to another business unit / product team
);

-- ── Closure lifecycle status ───────────────────────────────────────────────
CREATE TYPE closure_status AS ENUM (
  'draft',      -- analyst is composing
  'submitted',  -- sent to manager for review
  'approved',   -- manager/MLRO approved — triggers cases.status = 'closed'
  'rejected',   -- returned to analyst with reason
  'voided'      -- case was reopened; this closure record is historical
);

-- ── case_closures ──────────────────────────────────────────────────────────
-- One row per closure attempt. Preserved even when case is reopened (voided).
-- The approved closure is the authoritative decision record for regulators.
--
-- "Can an auditor reconstruct who decided, what disposition, what summary,
--  who approved, and when?" — YES, every field here answers that.
CREATE TABLE case_closures (
  id                      UUID                NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id                 UUID                NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  disposition             closure_disposition NOT NULL,
  summary                 TEXT                NOT NULL CHECK (length(trim(summary)) >= 100),
  reason_code             TEXT                NOT NULL DEFAULT '',
  -- STR linkage: if the case resulted in an STR filing, record it here
  filed_str               BOOLEAN             NOT NULL DEFAULT false,
  str_id                  UUID                REFERENCES str_filings(id) ON DELETE SET NULL,
  -- Risk rating at the time of closure (frozen snapshot)
  risk_rating_at_close    TEXT                CHECK (risk_rating_at_close IN ('low','medium','high','critical','unknown')),
  -- SAR consideration record: required when HIGH-risk case is not filed
  -- "I explicitly considered and rejected the SAR option" — PMLA requires this
  sar_considered          BOOLEAN             NOT NULL DEFAULT false,
  sar_rejection_reason    TEXT,
  -- Lifecycle actors
  created_by              UUID                NOT NULL REFERENCES users(id),
  reviewed_by             UUID                REFERENCES users(id),
  approved_by             UUID                REFERENCES users(id),
  -- Lifecycle timestamps
  submitted_at            TIMESTAMPTZ,
  reviewed_at             TIMESTAMPTZ,
  approved_at             TIMESTAMPTZ,
  -- Reopen tracking: when case is reopened, this record is voided (not deleted)
  voided_at               TIMESTAMPTZ,
  voided_by               UUID                REFERENCES users(id),
  voided_reason           TEXT,
  -- Status
  status                  closure_status      NOT NULL DEFAULT 'draft',
  created_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW()
  -- Note: NO deleted_at — closure records are NEVER deleted, only voided.
  -- Retention: must be kept for 5 years per PMLA Rule 3.
);

CREATE INDEX idx_closures_case    ON case_closures (case_id, created_at DESC);
CREATE INDEX idx_closures_status  ON case_closures (case_id, status);

CREATE TRIGGER case_closures_updated_at
  BEFORE UPDATE ON case_closures
  FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- ── Extend cases table with closure reference ──────────────────────────────
-- closure_id points to the currently active (approved) closure record.
-- closed_by / closed_at are denormalized for fast queries.
ALTER TABLE cases ADD COLUMN IF NOT EXISTS closure_id   UUID REFERENCES case_closures(id) ON DELETE SET NULL;
ALTER TABLE cases ADD COLUMN IF NOT EXISTS closed_by    UUID REFERENCES users(id);
ALTER TABLE cases ADD COLUMN IF NOT EXISTS closed_at    TIMESTAMPTZ;

-- ── Extend timeline event types for closure ────────────────────────────────
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'case_closure_submitted';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'case_closure_approved';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'case_closure_rejected';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'case_reopened';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE cases DROP COLUMN IF EXISTS closed_at;
ALTER TABLE cases DROP COLUMN IF EXISTS closed_by;
ALTER TABLE cases DROP COLUMN IF EXISTS closure_id;

DROP TABLE IF EXISTS case_closures;
DROP TYPE IF EXISTS closure_status;
DROP TYPE IF EXISTS closure_disposition;

-- +goose StatementEnd
