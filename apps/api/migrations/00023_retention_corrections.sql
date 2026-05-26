-- +goose Up
-- +goose StatementBegin

-- PMLA Section 12 mandates 10-year (3650-day) retention for all AML-related records.
-- Previous values (730, 2555, 365 days) were below the statutory minimum.
-- purge_enabled is forced to FALSE for all PMLA-protected resource types.

UPDATE data_retention_policies
   SET retention_days = 3650,
       purge_enabled  = FALSE,
       updated_at     = NOW()
 WHERE resource_type IN ('audit_logs', 'cases', 'documents', 'ai_logs');

-- STR filings were not in the original seed; PMLA requires 10-year retention.
-- purge_enabled=FALSE: these records must NEVER be automatically deleted.
INSERT INTO data_retention_policies (resource_type, retention_days, purge_enabled)
VALUES ('str_filings', 3650, FALSE)
ON CONFLICT (resource_type) DO UPDATE
   SET retention_days = 3650,
       purge_enabled  = FALSE,
       updated_at     = NOW();

-- Guard: prevent any future UPDATE from setting retention below the PMLA floor
-- for regulated resource types. This is a belt-and-suspenders DB-layer control.
CREATE OR REPLACE FUNCTION check_pmla_retention_floor()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.resource_type IN ('audit_logs', 'cases', 'documents', 'ai_logs', 'str_filings')
     AND NEW.retention_days < 3650
  THEN
    RAISE EXCEPTION 'PMLA_RETENTION_FLOOR: retention_days for % must be >= 3650 (10 years). Got %.',
      NEW.resource_type, NEW.retention_days;
  END IF;
  RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_pmla_retention_floor ON data_retention_policies;
CREATE TRIGGER trg_pmla_retention_floor
  BEFORE INSERT OR UPDATE ON data_retention_policies
  FOR EACH ROW EXECUTE FUNCTION check_pmla_retention_floor();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trg_pmla_retention_floor ON data_retention_policies;
DROP FUNCTION IF EXISTS check_pmla_retention_floor();

DELETE FROM data_retention_policies WHERE resource_type = 'str_filings';

UPDATE data_retention_policies
   SET retention_days = CASE resource_type
                          WHEN 'audit_logs' THEN 730
                          WHEN 'cases'      THEN 2555
                          WHEN 'documents'  THEN 2555
                          WHEN 'ai_logs'    THEN 365
                        END,
       updated_at     = NOW()
 WHERE resource_type IN ('audit_logs', 'cases', 'documents', 'ai_logs');

-- +goose StatementEnd
