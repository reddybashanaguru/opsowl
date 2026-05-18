-- SLA domain: configurable breach policies per priority level.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE sla_policies (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    priority     TEXT        NOT NULL UNIQUE,   -- 'critical'|'high'|'medium'|'low'
    warning_hours INT        NOT NULL,           -- Notify investigator at this elapsed hour
    breach_hours INT         NOT NULL,           -- SLA is breached after this many hours
    warn_pct     INT         NOT NULL DEFAULT 80, -- Warn when elapsed% >= this value
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Default seed data (applied in migration 00011):
-- critical: warn=3h,  breach=4h
-- high:     warn=20h, breach=24h
-- medium:   warn=58h, breach=72h
-- low:      warn=135h, breach=168h
