-- +goose Up
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'task_status_changed';

-- +goose Down
-- enum values cannot be removed in Postgres
