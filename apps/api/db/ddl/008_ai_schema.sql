-- AI domain: immutable interaction log, prompt registry, case insights.
-- Reference only. Do NOT run — use migrations/incremental/.

-- Versioned prompt registry — no hardcoded prompts in application code.
CREATE TABLE ai_prompts (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR      NOT NULL,
    slug           VARCHAR      NOT NULL,
    version        INT          NOT NULL,
    template       TEXT         NOT NULL,
    task_type      VARCHAR      NOT NULL,
    variables      JSONB        NOT NULL DEFAULT '[]',
    is_active      BOOL         NOT NULL DEFAULT true,
    ab_test_weight NUMERIC(3,2) NOT NULL DEFAULT 1.0,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (slug, version)
);

-- Immutable AI interaction log — UPDATE/DELETE blocked by trigger.
-- human_review_status state machine: pending → approved | rejected | edited
CREATE TABLE ai_logs (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id               UUID         NOT NULL REFERENCES cases(id),
    triggered_by          UUID         NOT NULL REFERENCES users(id),
    action_type           VARCHAR      NOT NULL CHECK (action_type IN (
                              'summarize', 'ask', 'generate_note', 'extract_actions', 'pattern_detect'
                          )),
    prompt_text           TEXT         NOT NULL,  -- PII-masked before storage
    response_text         TEXT         NOT NULL,
    model_used            VARCHAR      NOT NULL,
    provider              VARCHAR      NOT NULL,
    tokens_input          INT          NOT NULL,
    tokens_output         INT          NOT NULL,
    latency_ms            INT          NOT NULL,
    confidence_score      NUMERIC(3,2) CHECK (confidence_score BETWEEN 0 AND 1),
    citations             JSONB        NOT NULL DEFAULT '[]',
    human_review_status   VARCHAR      NOT NULL DEFAULT 'pending'
                              CHECK (human_review_status IN ('pending','approved','rejected','edited')),
    reviewed_by           UUID         REFERENCES users(id),
    reviewed_at           TIMESTAMPTZ,
    human_edited_response TEXT,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
    -- NO updated_at, NO deleted_at — ai_logs are append-only by design
);
CREATE INDEX idx_ai_logs_case   ON ai_logs (case_id, created_at DESC);
CREATE INDEX idx_ai_logs_review ON ai_logs (human_review_status, created_at);
CREATE INDEX idx_ai_logs_user   ON ai_logs (triggered_by, created_at DESC);
CREATE INDEX idx_ai_logs_action ON ai_logs (action_type, created_at DESC);

-- Per-case AI risk analysis results (Phase 2). Latest row is the live insight.
CREATE TABLE case_ai_insights (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id             UUID         NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    risk_score          NUMERIC(5,2),
    risk_level          TEXT GENERATED ALWAYS AS (
                            CASE
                                WHEN risk_score >= 75 THEN 'high'
                                WHEN risk_score >= 40 THEN 'medium'
                                ELSE 'low'
                            END
                        ) STORED,
    risk_factors        JSONB        NOT NULL DEFAULT '[]',
    entities            JSONB        NOT NULL DEFAULT '{}',
    anomalies           JSONB        NOT NULL DEFAULT '[]',
    recommended_actions JSONB        NOT NULL DEFAULT '[]',
    model_used          TEXT,
    confidence          NUMERIC(3,2),
    analysed_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_case_ai_insights_case_id  ON case_ai_insights(case_id);
CREATE INDEX idx_case_ai_insights_analysed ON case_ai_insights(case_id, analysed_at DESC);
