-- Core business tables for OpsOwl
-- This is a reference file. Do NOT run directly — use migrations/incremental/.

-- ── Organizations ─────────────────────────────────────────────────────────────
CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Users ─────────────────────────────────────────────────────────────────────
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT,
    full_name       TEXT NOT NULL,
    role            user_role NOT NULL DEFAULT 'analyst',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Cases ─────────────────────────────────────────────────────────────────────
CREATE TABLE cases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_number     TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    status          case_status NOT NULL DEFAULT 'draft',
    priority        case_priority NOT NULL DEFAULT 'medium',
    assigned_to     UUID REFERENCES users(id),
    created_by      UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL,
    sla_breached    BOOLEAN NOT NULL DEFAULT false,
    sla_breach_at   TIMESTAMPTZ,
    tags            TEXT[] NOT NULL DEFAULT '{}',
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cases_status       ON cases(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_priority     ON cases(priority) WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_assigned_to  ON cases(assigned_to) WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_created_by   ON cases(created_by) WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_sla_breached ON cases(sla_breached) WHERE deleted_at IS NULL;
CREATE INDEX idx_cases_created_at   ON cases(created_at DESC) WHERE deleted_at IS NULL;
