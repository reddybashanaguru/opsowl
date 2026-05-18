-- Communications domain: case communication logs and AI extractions.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE communication_logs (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id        UUID           NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    channel        comm_channel   NOT NULL,
    direction      comm_direction NOT NULL DEFAULT 'outbound',
    contact_name   TEXT           NOT NULL,
    contact_detail TEXT,
    subject        TEXT,
    body           TEXT           NOT NULL,
    outcome        TEXT,
    logged_by      UUID           NOT NULL REFERENCES users(id),
    occurred_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_comm_logs_case_id ON communication_logs(case_id, occurred_at DESC);

-- AI-extracted entities and action items from communication log body text.
CREATE TABLE comm_ai_extractions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    comm_log_id UUID        NOT NULL REFERENCES communication_logs(id) ON DELETE CASCADE,
    entities    JSONB       NOT NULL DEFAULT '{}',
    action_items JSONB      NOT NULL DEFAULT '[]',
    sentiment   TEXT,
    model_used  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_comm_ai_comm_log ON comm_ai_extractions (comm_log_id);
