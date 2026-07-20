-- +goose Up
-- +goose StatementBegin
ALTER TABLE csv_imports ADD COLUMN file_hash TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_csv_imports_file_hash ON csv_imports(file_hash) WHERE file_hash <> '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_csv_imports_file_hash;
ALTER TABLE csv_imports DROP COLUMN IF EXISTS file_hash;
-- +goose StatementEnd
