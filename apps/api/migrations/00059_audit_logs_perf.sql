-- +goose Up

-- Covering index for the infinite-scroll audit log query (resource_type + resource_id + created_at).
-- At 50M+ rows over 5 years, the existing idx_audit_resource doesn't include created_at,
-- forcing a sort step on every paginated load.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_resource_time
    ON audit_logs (resource_type, resource_id, created_at DESC)
    WHERE resource_id IS NOT NULL;

-- Covering index for the actor timeline (user activity feed).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_actor_time
    ON audit_logs (actor_id, created_at DESC);

-- seq_num is used for hash-chain verification; ensure it's a unique btree.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_seq_unique
    ON audit_logs (seq_num);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_resource_time;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_actor_time;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_seq_unique;
