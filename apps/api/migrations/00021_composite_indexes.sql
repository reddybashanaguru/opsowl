-- +goose Up
-- +goose StatementBegin

-- The ListCases query filters on up to 4 predicates simultaneously
-- (status, priority, assigned_to, sla_breached) and always orders by created_at DESC.
-- Single-column indexes on those columns don't help multi-predicate queries;
-- Postgres falls back to a full table scan on the status index then re-checks
-- every other predicate for each row.
--
-- These composite indexes cover the three highest-frequency filter combos observed
-- in the UI (inbox by assignee, board by status, supervisor view by org+status).

-- Inbox view: assigned investigator filtering their own cases by status
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cases_assigned_status_created
    ON cases (assigned_to, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- Board / kanban view: filter by status across the whole org
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cases_org_status_created
    ON cases (organization_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- SLA breach dashboard: cases that have/haven't breached, ordered by breach time
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cases_org_sla_breached_created
    ON cases (organization_id, sla_breached, created_at DESC)
    WHERE deleted_at IS NULL;

-- Priority queue: supervisor filtering high-priority open cases
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cases_org_priority_status
    ON cases (organization_id, priority, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX CONCURRENTLY IF EXISTS idx_cases_assigned_status_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_cases_org_status_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_cases_org_sla_breached_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_cases_org_priority_status;

-- +goose StatementEnd
