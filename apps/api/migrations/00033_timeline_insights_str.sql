-- +goose Up
-- +goose StatementBegin

ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'ai_insights_generated';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'str_draft_created';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'str_submitted';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'str_approved';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'str_rejected';
ALTER TYPE timeline_event_type ADD VALUE IF NOT EXISTS 'str_filed';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Enum values cannot be removed in Postgres; down migration is a no-op.
-- +goose StatementEnd
