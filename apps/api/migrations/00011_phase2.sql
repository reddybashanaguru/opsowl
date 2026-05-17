-- Phase 2 migrations

-- P2-6: Document category + retry tracking
ALTER TABLE documents ADD COLUMN IF NOT EXISTS doc_category TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS last_error_code TEXT;

-- P2-2: Case AI insights
CREATE TABLE IF NOT EXISTS case_ai_insights (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id         UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    risk_score      NUMERIC(5,2),
    risk_level      TEXT GENERATED ALWAYS AS (
                        CASE
                            WHEN risk_score >= 75 THEN 'high'
                            WHEN risk_score >= 40 THEN 'medium'
                            ELSE 'low'
                        END
                    ) STORED,
    risk_factors    JSONB NOT NULL DEFAULT '[]',
    entities        JSONB NOT NULL DEFAULT '{}',
    anomalies       JSONB NOT NULL DEFAULT '[]',
    recommended_actions JSONB NOT NULL DEFAULT '[]',
    model_used      TEXT,
    confidence      NUMERIC(3,2),
    analysed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_case_ai_insights_case_id ON case_ai_insights(case_id);
CREATE INDEX IF NOT EXISTS idx_case_ai_insights_analysed ON case_ai_insights(case_id, analysed_at DESC);

-- P2-4: Communication log
CREATE TYPE comm_channel AS ENUM ('email','call','field_visit','letter','whatsapp','sms','other');
CREATE TYPE comm_direction AS ENUM ('inbound','outbound');

CREATE TABLE IF NOT EXISTS communication_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id         UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    channel         comm_channel NOT NULL,
    direction       comm_direction NOT NULL DEFAULT 'outbound',
    contact_name    TEXT NOT NULL,
    contact_detail  TEXT,
    subject         TEXT,
    body            TEXT NOT NULL,
    outcome         TEXT,
    logged_by       UUID NOT NULL REFERENCES users(id),
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_comm_logs_case_id ON communication_logs(case_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS comm_ai_extractions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    comm_log_id     UUID NOT NULL REFERENCES communication_logs(id) ON DELETE CASCADE,
    entities        JSONB NOT NULL DEFAULT '{}',
    action_items    JSONB NOT NULL DEFAULT '[]',
    sentiment       TEXT,
    model_used      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- P2-3: SLA policies
CREATE TABLE IF NOT EXISTS sla_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    priority        TEXT NOT NULL UNIQUE,
    warning_hours   INT NOT NULL,
    breach_hours    INT NOT NULL,
    warn_pct        INT NOT NULL DEFAULT 80,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default SLA policies matching existing hardcoded values
INSERT INTO sla_policies (priority, warning_hours, breach_hours, warn_pct) VALUES
    ('critical', 3,   4,   80),
    ('high',     20,  24,  80),
    ('medium',   58,  72,  80),
    ('low',      135, 168, 80)
ON CONFLICT (priority) DO NOTHING;

-- P2-5: Workload view (pure query — no new tables needed)
-- Dashboard analytics index
CREATE INDEX IF NOT EXISTS idx_cases_assigned_status ON cases(assigned_to, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_assigned_status ON tasks(assigned_to, status) WHERE deleted_at IS NULL;
