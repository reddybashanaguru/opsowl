-- +goose Up
-- +goose StatementBegin

-- Add suspicion_date to str_filings.
-- Federal Decree-Law No. 20 of 2018 Art. 15: the 30-day filing clock starts
-- when suspicion first arose, not when the STR was created in the system.
ALTER TABLE str_filings
  ADD COLUMN IF NOT EXISTS suspicion_date DATE;

-- Backfill: use created_at as best approximation for existing rows.
UPDATE str_filings SET suspicion_date = created_at::DATE WHERE suspicion_date IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE str_filings DROP COLUMN IF EXISTS suspicion_date;
-- +goose StatementEnd
