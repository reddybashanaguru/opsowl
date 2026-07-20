-- +goose Up
-- +goose StatementBegin

-- CHECK constraints: prevent invalid status values being written to either table.
ALTER TABLE csv_imports
  ADD CONSTRAINT csv_imports_status_check
  CHECK (status IN ('pending','processing','done','partial','failed'));

ALTER TABLE csv_import_rows
  ADD CONSTRAINT csv_import_rows_status_check
  CHECK (status IN ('pending','success','failed'));

-- Composite index: ListImportRows filters on (import_id, status).
-- Without this, filtering by status requires a full scan of all rows
-- for a given import (up to 5,000 rows per job).
CREATE INDEX idx_csv_import_rows_import_status
  ON csv_import_rows(import_id, status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_csv_import_rows_import_status;
ALTER TABLE csv_import_rows DROP CONSTRAINT IF EXISTS csv_import_rows_status_check;
ALTER TABLE csv_imports     DROP CONSTRAINT IF EXISTS csv_imports_status_check;
-- +goose StatementEnd
