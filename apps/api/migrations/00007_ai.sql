-- +goose Up

-- Versioned prompt registry (no hard-coded prompts in code)
CREATE TABLE ai_prompts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR     NOT NULL,
    slug           VARCHAR     NOT NULL,
    version        INT         NOT NULL,
    template       TEXT        NOT NULL,
    task_type      VARCHAR     NOT NULL,
    variables      JSONB       NOT NULL DEFAULT '[]',
    is_active      BOOL        NOT NULL DEFAULT true,
    ab_test_weight NUMERIC(3,2) NOT NULL DEFAULT 1.0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (slug, version)
);

CREATE TRIGGER ai_prompts_updated_at
    BEFORE UPDATE ON ai_prompts
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();

-- AI interaction log — immutable (no deleted_at, no UPDATE except review fields)
CREATE TABLE ai_logs (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id               UUID        NOT NULL REFERENCES cases(id),
    triggered_by          UUID        NOT NULL REFERENCES users(id),
    action_type           VARCHAR     NOT NULL CHECK (action_type IN (
                              'summarize', 'ask', 'generate_note', 'extract_actions', 'pattern_detect'
                          )),
    prompt_text           TEXT        NOT NULL,
    response_text         TEXT        NOT NULL,
    model_used            VARCHAR     NOT NULL,
    provider              VARCHAR     NOT NULL,
    tokens_input          INT         NOT NULL,
    tokens_output         INT         NOT NULL,
    latency_ms            INT         NOT NULL,
    confidence_score      NUMERIC(3,2) CHECK (confidence_score BETWEEN 0 AND 1),
    citations             JSONB       NOT NULL DEFAULT '[]',
    human_review_status   VARCHAR     NOT NULL DEFAULT 'pending'
                              CHECK (human_review_status IN ('pending','approved','rejected','edited')),
    reviewed_by           UUID        REFERENCES users(id),
    reviewed_at           TIMESTAMPTZ,
    human_edited_response TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- NO updated_at, NO deleted_at — ai_logs are immutable append-only
);

CREATE INDEX idx_ai_logs_case   ON ai_logs (case_id, created_at DESC);
CREATE INDEX idx_ai_logs_review ON ai_logs (human_review_status, created_at);
CREATE INDEX idx_ai_logs_user   ON ai_logs (triggered_by, created_at DESC);
CREATE INDEX idx_ai_logs_action ON ai_logs (action_type, created_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_ai_logs_action;
DROP INDEX IF EXISTS idx_ai_logs_user;
DROP INDEX IF EXISTS idx_ai_logs_review;
DROP INDEX IF EXISTS idx_ai_logs_case;
DROP TABLE IF EXISTS ai_logs;

DROP TRIGGER IF EXISTS ai_prompts_updated_at ON ai_prompts;
DROP TABLE IF EXISTS ai_prompts;
