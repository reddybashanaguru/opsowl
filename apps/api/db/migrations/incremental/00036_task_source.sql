-- +goose Up
-- +goose StatementBegin
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS template_id UUID REFERENCES case_templates(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks(template_id) WHERE template_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tasks_template_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS template_id;
-- +goose StatementEnd
