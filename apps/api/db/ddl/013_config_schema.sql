-- Config domain: runtime system configuration (single-row JSONB pattern).
-- Reference only. Do NOT run — use migrations/incremental/.

-- Stores all runtime config as a single JSONB row.
-- Avoids EAV anti-pattern while keeping schema flexible for new config keys.
CREATE TABLE system_config (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    config     JSONB       NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID        REFERENCES users(id)
);
-- Enforced single-row: INSERT is only allowed once via application bootstrap.
-- Keys: app_name, industry, and 14 label-pair fields (see config module).
