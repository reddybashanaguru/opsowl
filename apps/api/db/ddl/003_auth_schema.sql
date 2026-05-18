-- Authentication domain: users, sessions, tokens, audit, MFA, SSE tickets.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    full_name     TEXT        NOT NULL,
    role          user_role   NOT NULL DEFAULT 'analyst',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);
CREATE INDEX idx_users_email     ON users (email)     WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role      ON users (role)      WHERE deleted_at IS NULL;
CREATE INDEX idx_users_is_active ON users (is_active) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users (created_at);

CREATE TABLE refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rt_user_id    ON refresh_tokens (user_id);
CREATE INDEX idx_rt_expires_at ON refresh_tokens (expires_at);
CREATE INDEX idx_rt_hash       ON refresh_tokens (token_hash) WHERE revoked_at IS NULL;

CREATE TABLE sessions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ip_address     INET        NOT NULL,
    user_agent     TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sessions_user_id     ON sessions (user_id);
CREATE INDEX idx_sessions_last_active ON sessions (last_active_at);

CREATE TABLE audit_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_code    TEXT        NOT NULL,
    actor_id      UUID        REFERENCES users(id) ON DELETE SET NULL,
    ip_address    TEXT,
    resource_type TEXT,
    resource_id   UUID,
    metadata      JSONB       NOT NULL DEFAULT '{}',
    severity      TEXT        NOT NULL DEFAULT 'info',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- Immutable: UPDATE/DELETE blocked by trigger prevent_audit_modification()
);
CREATE INDEX idx_audit_event_code ON audit_logs (event_code);
CREATE INDEX idx_audit_actor_id   ON audit_logs (actor_id);
CREATE INDEX idx_audit_created_at ON audit_logs (created_at);
CREATE INDEX idx_audit_resource   ON audit_logs (resource_type, resource_id) WHERE resource_id IS NOT NULL;

-- Short-lived tokens for SSE stream authentication.
-- Prevents full 1-hour access JWTs from appearing in nginx access logs.
CREATE TABLE sse_tickets (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sse_tickets_hash    ON sse_tickets (token_hash);
CREATE INDEX idx_sse_tickets_user    ON sse_tickets (user_id);
CREATE INDEX idx_sse_tickets_expires ON sse_tickets (expires_at);

-- TOTP / MFA secrets per user. One active secret per user at a time.
CREATE TABLE totp_secrets (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret      TEXT        NOT NULL,           -- Base32-encoded TOTP secret
    status      totp_status NOT NULL DEFAULT 'pending',
    verified_at TIMESTAMPTZ,                    -- Set when user completes enrollment
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_totp_user_active ON totp_secrets (user_id) WHERE status = 'active';
CREATE INDEX idx_totp_user_id ON totp_secrets (user_id);

-- Single-use backup codes for MFA recovery (8 codes per enrollment).
CREATE TABLE mfa_backup_codes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT        NOT NULL,            -- SHA-256 hex of the raw code
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backup_codes_user ON mfa_backup_codes (user_id) WHERE used_at IS NULL;
