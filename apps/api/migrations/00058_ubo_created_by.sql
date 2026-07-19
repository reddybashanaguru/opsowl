-- +goose Up
ALTER TABLE ubo_records ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id);

-- +goose Down
ALTER TABLE ubo_records DROP COLUMN IF EXISTS created_by;
