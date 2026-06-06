-- +goose Up
-- +goose StatementBegin

-- ── Extend ai_logs to include analysis actions ─────────────────────────────
-- The original CHECK constraint is too narrow; we need 'analyse_case' for
-- the insights handler to log its AI calls with full auditability.
ALTER TABLE ai_logs DROP CONSTRAINT IF EXISTS ai_logs_action_type_check;
ALTER TABLE ai_logs ADD CONSTRAINT ai_logs_action_type_check
  CHECK (action_type IN (
    'summarize', 'ask', 'generate_note', 'extract_actions', 'pattern_detect',
    'analyse_case', 'generate_narrative'
  ));

-- ── Finding categories ─────────────────────────────────────────────────────
CREATE TYPE finding_category AS ENUM (
  'structuring',
  'layering',
  'integration',
  'smurfing',
  'round_tripping',
  'pep_exposure',
  'sanctions_evasion',
  'unusual_cash',
  'rapid_movement',
  'identity_fraud',
  'account_takeover',
  'other'
);

-- ── Finding severity ───────────────────────────────────────────────────────
CREATE TYPE finding_severity AS ENUM (
  'low',
  'medium',
  'high',
  'critical'
);

-- ── Finding lifecycle status ───────────────────────────────────────────────
CREATE TYPE finding_status AS ENUM (
  'draft',
  'under_review',
  'approved',
  'rejected',
  'contested',
  'archived'
);

-- ── Narrative weight for STR/closure auto-population ──────────────────────
CREATE TYPE finding_narrative_weight AS ENUM (
  'primary',
  'supporting',
  'contextual'
);

-- ── case_findings ──────────────────────────────────────────────────────────
-- One row per formal investigative finding on a case.
-- These are the structured conclusions an investigator reaches — separate
-- from unstructured notes. An MLRO reviews approved findings before signing
-- off on case closure.
CREATE TABLE case_findings (
  id                  UUID                      PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id             UUID                      NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  title               TEXT                      NOT NULL CHECK (length(trim(title)) > 0),
  description         TEXT                      NOT NULL CHECK (length(trim(description)) > 0),
  category            finding_category          NOT NULL,
  severity            finding_severity          NOT NULL DEFAULT 'medium',
  status              finding_status            NOT NULL DEFAULT 'draft',
  narrative_weight    finding_narrative_weight  NOT NULL DEFAULT 'supporting',
  confidence_score    NUMERIC(3,2)              CHECK (confidence_score BETWEEN 0 AND 1),
  ai_generated        BOOLEAN                   NOT NULL DEFAULT false,
  source_ai_log_id    UUID                      REFERENCES ai_logs(id) ON DELETE SET NULL,
  created_by          UUID                      NOT NULL REFERENCES users(id),
  reviewed_by         UUID                      REFERENCES users(id),
  approved_by         UUID                      REFERENCES users(id),
  -- severity dispute tracking: when manager downgrades CRITICAL→HIGH they must explain
  severity_original   finding_severity,
  severity_disputed_by UUID                     REFERENCES users(id),
  severity_override_reason TEXT,
  -- review timestamps
  reviewed_at         TIMESTAMPTZ,
  approved_at         TIMESTAMPTZ,
  created_at          TIMESTAMPTZ               NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ               NOT NULL DEFAULT NOW(),
  deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_findings_case   ON case_findings (case_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_findings_status ON case_findings (case_id, status)          WHERE deleted_at IS NULL;
CREATE INDEX idx_findings_sev    ON case_findings (case_id, severity)        WHERE deleted_at IS NULL;

CREATE TRIGGER case_findings_updated_at
  BEFORE UPDATE ON case_findings
  FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- ── finding_evidence_links ─────────────────────────────────────────────────
-- Polymorphic association: links a finding to any type of evidence item.
-- evidence_type is validated at the application layer.
-- The combination (finding_id, evidence_type, evidence_id) is unique to
-- prevent double-linking the same piece of evidence to a finding.
CREATE TABLE finding_evidence_links (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  finding_id      UUID        NOT NULL REFERENCES case_findings(id) ON DELETE CASCADE,
  evidence_type   TEXT        NOT NULL CHECK (evidence_type IN (
                    'document', 'note', 'transaction', 'ai_log', 'subject', 'alert'
                  )),
  evidence_id     UUID        NOT NULL,
  -- For document evidence: optional page/chunk reference for precise citation
  page_number     INTEGER,
  chunk_id        UUID,       -- references document_chunks(id) at app level
  citation        TEXT,       -- human-readable excerpt: "Page 3, row 7 — wire transfer of ₹45L"
  relevance_note  TEXT,       -- why this evidence supports the finding
  linked_by       UUID        NOT NULL REFERENCES users(id),
  linked_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (finding_id, evidence_type, evidence_id)
);

CREATE INDEX idx_evidence_links_finding  ON finding_evidence_links (finding_id);
CREATE INDEX idx_evidence_links_evidence ON finding_evidence_links (evidence_type, evidence_id);

-- ── Extend timeline event types for findings ───────────────────────────────
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'finding_created';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'finding_status_changed';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'finding_evidence_linked';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'finding_evidence_removed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS finding_evidence_links;
DROP TABLE IF EXISTS case_findings;
DROP TYPE IF EXISTS finding_narrative_weight;
DROP TYPE IF EXISTS finding_status;
DROP TYPE IF EXISTS finding_severity;
DROP TYPE IF EXISTS finding_category;

-- Restore original ai_logs constraint
ALTER TABLE ai_logs DROP CONSTRAINT IF EXISTS ai_logs_action_type_check;
ALTER TABLE ai_logs ADD CONSTRAINT ai_logs_action_type_check
  CHECK (action_type IN (
    'summarize', 'ask', 'generate_note', 'extract_actions', 'pattern_detect'
  ));

-- +goose StatementEnd
