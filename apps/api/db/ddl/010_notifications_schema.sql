-- Notifications domain: in-app bell notifications and user preferences.
-- Reference only. Do NOT run — use migrations/incremental/.

CREATE TABLE notifications (
    id           UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id UUID                  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         notification_type     NOT NULL,
    priority     notification_priority NOT NULL DEFAULT 'medium',
    title        TEXT                  NOT NULL,
    message      TEXT                  NOT NULL,
    link_url     TEXT,
    case_id      UUID                  REFERENCES cases(id) ON DELETE SET NULL,
    is_read      BOOLEAN               NOT NULL DEFAULT FALSE,
    read_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ           NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notif_recipient    ON notifications (recipient_id, created_at DESC);
CREATE INDEX idx_notif_unread       ON notifications (recipient_id) WHERE is_read = FALSE;
CREATE INDEX idx_notif_case_id      ON notifications (case_id) WHERE case_id IS NOT NULL;

CREATE TABLE notification_preferences (
    id           UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID                  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type         notification_type     NOT NULL,
    in_app       BOOLEAN               NOT NULL DEFAULT TRUE,
    email        BOOLEAN               NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, type)
);
CREATE INDEX idx_notif_pref_user ON notification_preferences (user_id);
