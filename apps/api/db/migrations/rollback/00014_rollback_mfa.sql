-- Emergency rollback for 00014_mfa.sql
DROP TABLE IF EXISTS mfa_backup_codes;
DROP TABLE IF EXISTS totp_secrets;
DROP TYPE IF EXISTS totp_status;
