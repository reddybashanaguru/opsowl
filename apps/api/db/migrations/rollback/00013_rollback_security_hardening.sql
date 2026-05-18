-- Emergency rollback for 00013_security_hardening.sql
-- Run ONLY for disaster recovery. Removes audit immutability protection.

DROP INDEX IF EXISTS idx_sse_tickets_expires;
DROP INDEX IF EXISTS idx_sse_tickets_user;
DROP INDEX IF EXISTS idx_sse_tickets_hash;
DROP TABLE IF EXISTS sse_tickets;

DROP TRIGGER IF EXISTS audit_logs_immutable ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_modification();

ALTER TABLE documents DROP COLUMN IF EXISTS file_sha256;
