-- +goose Up

CREATE TYPE notification_type AS ENUM (
    'case_assigned',
    'case_status_changed',
    'case_sla_warning',
    'case_sla_breached',
    'task_assigned',
    'task_due_soon',
    'task_overdue',
    'mention',
    'ai_review_required',
    'document_processed',
    'comment_added'
);

CREATE TYPE notification_priority AS ENUM ('low', 'medium', 'high', 'critical');

CREATE TABLE notifications (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         notification_type NOT NULL,
    priority     notification_priority NOT NULL DEFAULT 'medium',
    title        TEXT        NOT NULL,
    message      TEXT        NOT NULL,
    link_url     TEXT,
    is_read      BOOLEAN     NOT NULL DEFAULT FALSE,
    read_at      TIMESTAMPTZ,
    case_id      UUID        REFERENCES cases(id) ON DELETE SET NULL,
    task_id      UUID        REFERENCES tasks(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient_unread  ON notifications(recipient_id, is_read, created_at DESC);
CREATE INDEX idx_notifications_recipient_created ON notifications(recipient_id, created_at DESC);
CREATE INDEX idx_notifications_case              ON notifications(case_id) WHERE case_id IS NOT NULL;

CREATE TABLE notification_preferences (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type notification_type NOT NULL,
    in_app_enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    email_enabled     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, notification_type)
);

CREATE INDEX idx_notification_prefs_user ON notification_preferences(user_id);

-- +goose Down

DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
DROP TYPE IF EXISTS notification_priority;
DROP TYPE IF EXISTS notification_type;
