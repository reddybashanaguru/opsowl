-- name: CreateNotification :one
INSERT INTO notifications (recipient_id, type, priority, title, message, link_url, case_id, task_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, recipient_id, type, priority, title, message, link_url, is_read, read_at, case_id, task_id, created_at, updated_at;

-- name: ListNotifications :many
SELECT id, recipient_id, type, priority, title, message, link_url, is_read, read_at, case_id, task_id, created_at, updated_at
FROM notifications
WHERE recipient_id = $1
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR id < $2)
ORDER BY created_at DESC
LIMIT $3;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1 AND is_read = FALSE;

-- name: MarkNotificationRead :one
UPDATE notifications SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
WHERE id = $1 AND recipient_id = $2
RETURNING id;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET is_read = TRUE, read_at = NOW(), updated_at = NOW()
WHERE recipient_id = $1 AND is_read = FALSE;

-- name: GetNotificationPreference :one
SELECT id, user_id, notification_type, in_app_enabled, email_enabled
FROM notification_preferences
WHERE user_id = $1 AND notification_type = $2
LIMIT 1;

-- name: UpsertNotificationPreference :one
INSERT INTO notification_preferences (user_id, notification_type, in_app_enabled, email_enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, notification_type)
DO UPDATE SET in_app_enabled = $3, email_enabled = $4, updated_at = NOW()
RETURNING id, user_id, notification_type, in_app_enabled, email_enabled, created_at, updated_at;

-- name: ListNotificationPreferences :many
SELECT id, user_id, notification_type, in_app_enabled, email_enabled
FROM notification_preferences
WHERE user_id = $1
ORDER BY notification_type;
