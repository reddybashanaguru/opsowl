-- +goose Up
-- Add DFAT (Australian autonomous sanctions) to the sanctions_lists source constraint.
-- DFAT = Department of Foreign Affairs and Trade, Australia.
-- Required for AUSTRAC-regulated entities screening against Australian autonomous sanctions.

-- Rename the column-level check constraint to a named table constraint so it can be dropped cleanly.
ALTER TABLE sanctions_lists DROP CONSTRAINT IF EXISTS sanctions_lists_source_check;
ALTER TABLE sanctions_lists ADD CONSTRAINT sanctions_lists_source_check
    CHECK (source IN ('eu_consolidated','un_consolidated','ofac_sdn','dfat_consolidated'));

-- +goose Down
ALTER TABLE sanctions_lists DROP CONSTRAINT IF EXISTS sanctions_lists_source_check;
ALTER TABLE sanctions_lists ADD CONSTRAINT sanctions_lists_source_check
    CHECK (source IN ('eu_consolidated','un_consolidated','ofac_sdn'));
