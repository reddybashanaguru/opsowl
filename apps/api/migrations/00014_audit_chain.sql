-- +goose Up
-- +goose StatementBegin

-- Hash-chain audit log: each row stores a SHA-256 hash that covers the content
-- of the row plus the hash of the previous row. Deleting or modifying any row
-- breaks the chain and is detectable via the /api/v1/audit/verify endpoint.
--
-- seq_num: monotonically increasing sequence (gap = deletion detected)
-- prev_hash: hash of (seq_num-1) row, empty string for seq_num=1
-- entry_hash: SHA-256( seq_num || event_code || actor_id || created_at || metadata::text || prev_hash )
--             computed by the application on insert

ALTER TABLE audit_logs
  ADD COLUMN IF NOT EXISTS seq_num    BIGINT UNIQUE,
  ADD COLUMN IF NOT EXISTS prev_hash  TEXT   NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS entry_hash TEXT   NOT NULL DEFAULT '';

-- Sequence for monotonic ordering independent of UUID ordering
CREATE SEQUENCE IF NOT EXISTS audit_log_seq START 1 INCREMENT 1;

-- Backfill existing rows with sequential numbers (no real hash chain — mark as pre-chain)
DO $$
DECLARE
  r RECORD;
  n BIGINT := 1;
BEGIN
  FOR r IN SELECT id FROM audit_logs ORDER BY created_at, id LOOP
    UPDATE audit_logs SET seq_num = n, prev_hash = 'genesis', entry_hash = 'pre-chain'
    WHERE id = r.id;
    n := n + 1;
  END LOOP;
  -- Advance the sequence past existing rows
  PERFORM setval('audit_log_seq', GREATEST(n, 1));
END $$;

CREATE INDEX IF NOT EXISTS idx_audit_seq_num ON audit_logs (seq_num);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SEQUENCE IF EXISTS audit_log_seq;
ALTER TABLE audit_logs
  DROP COLUMN IF EXISTS seq_num,
  DROP COLUMN IF EXISTS prev_hash,
  DROP COLUMN IF EXISTS entry_hash;
-- +goose StatementEnd
