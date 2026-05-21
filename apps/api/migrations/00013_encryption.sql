-- +goose Up
-- +goose StatementBegin

-- Enable pgcrypto for column-level AES-256 encryption of PII fields.
-- OS-level volume encryption (LUKS/dm-crypt) handles raw storage;
-- pgcrypto provides defence-in-depth for database dumps and cold backups.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Encryption key is loaded from the ENCRYPTION_KEY env var at runtime via
-- the encrypt_pii / decrypt_pii helper functions defined in app code.
-- We store a key_id in each row so that key rotation is traceable.
ALTER TABLE cases
  ADD COLUMN IF NOT EXISTS encrypted_description TEXT,
  ADD COLUMN IF NOT EXISTS description_key_id     TEXT;

ALTER TABLE notes
  ADD COLUMN IF NOT EXISTS encrypted_content  TEXT,
  ADD COLUMN IF NOT EXISTS content_key_id     TEXT;

ALTER TABLE communication_logs
  ADD COLUMN IF NOT EXISTS encrypted_summary  TEXT,
  ADD COLUMN IF NOT EXISTS summary_key_id     TEXT;

-- MinIO SSE master key is configured externally (MINIO_KMS_SECRET_KEY env var).
-- This table records when the key was last rotated so compliance reports have an audit trail.
CREATE TABLE IF NOT EXISTS encryption_key_rotations (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  key_id        TEXT        NOT NULL,
  rotated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  rotated_by    UUID        REFERENCES users(id) ON DELETE SET NULL,
  notes         TEXT
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS encryption_key_rotations;
ALTER TABLE communication_logs DROP COLUMN IF EXISTS encrypted_summary, DROP COLUMN IF EXISTS summary_key_id;
ALTER TABLE notes DROP COLUMN IF EXISTS encrypted_content, DROP COLUMN IF EXISTS content_key_id;
ALTER TABLE cases DROP COLUMN IF EXISTS encrypted_description, DROP COLUMN IF EXISTS description_key_id;
-- +goose StatementEnd
