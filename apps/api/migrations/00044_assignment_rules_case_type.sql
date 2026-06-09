-- +goose Up
-- +goose StatementBegin
ALTER TABLE assignment_rules ADD COLUMN IF NOT EXISTS match_case_type TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE assignment_rules DROP COLUMN IF EXISTS match_case_type;
-- +goose StatementEnd
