-- All PostgreSQL enum types used across the schema.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TYPE user_role AS ENUM (
    'viewer', 'analyst', 'manager', 'admin', 'super_admin'
);

CREATE TYPE case_status AS ENUM (
    'draft', 'open', 'in_progress', 'pending_review', 'escalated', 'closed'
);

CREATE TYPE case_priority AS ENUM (
    'low', 'medium', 'high', 'critical'
);

CREATE TYPE document_processing_status AS ENUM (
    'pending', 'processing', 'completed', 'failed'
);

CREATE TYPE timeline_event_source AS ENUM (
    'system', 'user', 'ai'
);

CREATE TYPE notification_type AS ENUM (
    'case_assigned', 'case_status_changed', 'case_sla_warning', 'case_escalated',
    'task_assigned', 'task_due_soon', 'task_overdue', 'task_completed',
    'note_mention', 'document_processed', 'ai_review_required'
);

CREATE TYPE notification_priority AS ENUM (
    'low', 'medium', 'high', 'urgent'
);

CREATE TYPE comm_channel AS ENUM (
    'email', 'call', 'field_visit', 'letter', 'whatsapp', 'sms', 'other'
);

CREATE TYPE comm_direction AS ENUM (
    'inbound', 'outbound'
);

CREATE TYPE totp_status AS ENUM (
    'pending', 'active', 'disabled'
);
