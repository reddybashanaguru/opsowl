-- Default SLA policy seeds for OpsOwl
-- Run after migrations to populate initial SLA thresholds.
-- Safe to re-run (upserts on priority conflict).

INSERT INTO sla_policies (priority, warning_hours, breach_hours, warn_pct)
VALUES
    ('critical', 2,  4,  80),
    ('high',     6,  12, 80),
    ('medium',   18, 24, 80),
    ('low',      48, 72, 80)
ON CONFLICT (priority) DO UPDATE
    SET warning_hours = EXCLUDED.warning_hours,
        breach_hours  = EXCLUDED.breach_hours,
        warn_pct      = EXCLUDED.warn_pct,
        updated_at    = NOW();
