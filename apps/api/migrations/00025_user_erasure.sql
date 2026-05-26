-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS erased_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS erased_at;
