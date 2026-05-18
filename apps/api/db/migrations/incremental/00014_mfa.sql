-- +goose Up
CREATE TYPE totp_status AS ENUM ('pending', 'active', 'revoked');

CREATE TABLE totp_secrets (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret      TEXT        NOT NULL,
    status      totp_status NOT NULL DEFAULT 'pending',
    verified_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_totp_user_active ON totp_secrets (user_id) WHERE status = 'active';
CREATE INDEX idx_totp_user_id ON totp_secrets (user_id);

CREATE TABLE mfa_backup_codes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT        NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_backup_codes_user ON mfa_backup_codes (user_id) WHERE used_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS mfa_backup_codes;
DROP TABLE IF EXISTS totp_secrets;
DROP TYPE IF EXISTS totp_status;
