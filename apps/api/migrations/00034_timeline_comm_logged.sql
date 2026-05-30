-- +goose Up
-- +goose StatementBegin
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'comm_logged';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Enum values cannot be removed in Postgres; down migration is a no-op.
-- +goose StatementEnd
