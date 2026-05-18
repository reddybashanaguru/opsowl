-- name: InsertTimelineEvent :one
INSERT INTO timeline_events (case_id, event_type, source, actor_id, occurred_at, payload, metadata)
VALUES (@case_id, @event_type, @source, @actor_id, @occurred_at, @payload, @metadata)
RETURNING *;

-- name: ListTimelineEvents :many
SELECT
  t.*,
  u.full_name AS actor_name,
  u.email     AS actor_email
FROM timeline_events t
LEFT JOIN users u ON u.id = t.actor_id
WHERE t.case_id    = @case_id
  AND t.deleted_at IS NULL
  AND (sqlc.narg('event_type')::timeline_event_type IS NULL OR t.event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('source')::timeline_event_source   IS NULL OR t.source     = sqlc.narg('source'))
  AND (sqlc.narg('actor_id')::uuid                  IS NULL OR t.actor_id   = sqlc.narg('actor_id'))
  AND (sqlc.narg('from_time')::timestamptz           IS NULL OR t.occurred_at >= sqlc.narg('from_time'))
  AND (sqlc.narg('to_time')::timestamptz             IS NULL OR t.occurred_at <= sqlc.narg('to_time'))
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR t.occurred_at < sqlc.narg('cursor_time')
    OR (t.occurred_at = sqlc.narg('cursor_time') AND t.id < sqlc.narg('cursor_id')::uuid)
  )
ORDER BY t.occurred_at DESC, t.id DESC
LIMIT @row_limit;

-- name: CountTimelineEvents :one
SELECT COUNT(*) FROM timeline_events
WHERE case_id    = @case_id
  AND deleted_at IS NULL
  AND (sqlc.narg('event_type')::timeline_event_type IS NULL OR event_type = sqlc.narg('event_type'))
  AND (sqlc.narg('source')::timeline_event_source   IS NULL OR source     = sqlc.narg('source'))
  AND (sqlc.narg('actor_id')::uuid                  IS NULL OR actor_id   = sqlc.narg('actor_id'))
  AND (sqlc.narg('from_time')::timestamptz           IS NULL OR occurred_at >= sqlc.narg('from_time'))
  AND (sqlc.narg('to_time')::timestamptz             IS NULL OR occurred_at <= sqlc.narg('to_time'));

-- name: GetTimelineEventByID :one
SELECT t.*, u.full_name AS actor_name, u.email AS actor_email
FROM timeline_events t
LEFT JOIN users u ON u.id = t.actor_id
WHERE t.id = $1 AND t.deleted_at IS NULL;
