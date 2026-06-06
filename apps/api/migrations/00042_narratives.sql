-- +goose Up
-- +goose StatementBegin

-- ── AI Investigation Narrative ────────────────────────────────────────────────
-- Stores AI-generated investigation narratives (STR/SAR suspicion section,
-- case summary, regulatory submission text). Each generation is immutable
-- once approved — referenced by ai_log_id for full auditability.
CREATE TABLE case_narratives (
  id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id           UUID        NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  narrative_type    TEXT        NOT NULL DEFAULT 'investigation_summary',
  -- Types:
  --   investigation_summary  — overall narrative of what happened and why it's suspicious
  --   str_suspicion_basis    — text for the suspicion_basis field of an STR
  --   sar_narrative          — SAR filing narrative
  --   closure_summary        — auto-drafted summary for the closure form
  status            TEXT        NOT NULL DEFAULT 'draft',
  -- draft → approved | rejected
  body              TEXT        NOT NULL DEFAULT '',  -- the narrative text
  body_edited       TEXT,                             -- human-edited version (if analyst revised it)
  -- Audit fields
  generated_by      UUID        REFERENCES users(id) ON DELETE SET NULL,  -- who triggered generation
  ai_log_id         UUID        REFERENCES ai_logs(id) ON DELETE SET NULL, -- full AI interaction record
  reviewed_by       UUID        REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at       TIMESTAMPTZ,
  rejection_reason  TEXT        NOT NULL DEFAULT '',
  -- Generation params snapshot (for reproducibility)
  model_used        TEXT        NOT NULL DEFAULT '',
  prompt_version    TEXT        NOT NULL DEFAULT 'v1',
  findings_included INT         NOT NULL DEFAULT 0,  -- how many findings were fed to the LLM
  txns_included     INT         NOT NULL DEFAULT 0,  -- how many transactions were fed
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_narratives_case    ON case_narratives(case_id);
CREATE INDEX idx_narratives_status  ON case_narratives(case_id, status);
CREATE INDEX idx_narratives_type    ON case_narratives(case_id, narrative_type);

-- Timeline event types for narrative lifecycle
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'narrative_generated';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'narrative_approved';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'narrative_rejected';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'narrative_edited';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS case_narratives;
-- +goose StatementEnd
