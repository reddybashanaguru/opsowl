-- +goose Up
-- EU FATF typology indicator codes for goAML XML EU submissions.
-- Auto-detected from suspicion_basis text, but compliance officers can
-- override/extend with jurisdiction-specific codes (e.g. ML01, TF02, etc.)
ALTER TABLE str_filings
    ADD COLUMN IF NOT EXISTS eu_fatf_indicators TEXT[] NOT NULL DEFAULT '{}';

-- Country context for multi-jurisdiction deployments
ALTER TABLE str_filings
    ADD COLUMN IF NOT EXISTS submission_country TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE str_filings DROP COLUMN IF EXISTS eu_fatf_indicators;
ALTER TABLE str_filings DROP COLUMN IF EXISTS submission_country;
