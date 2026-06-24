-- +goose Up
-- Fuzzy Entity Resolution: phone-array GIN index for overlap matching.
-- pg_trgm and idx_subjects_name (canonical_name gin_trgm_ops) already exist (migration 00036).
-- entity_relationships table already exists (migration 00038) and accepts type SAME_ENTITY.
CREATE INDEX IF NOT EXISTS idx_subjects_phones_gin ON subjects USING gin (phones);

-- +goose Down
DROP INDEX IF EXISTS idx_subjects_phones_gin;
