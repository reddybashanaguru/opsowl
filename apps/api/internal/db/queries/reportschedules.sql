-- ── Report Schedules ───────────────────────────────────────────────────────────

-- name: InsertReportSchedule :one
INSERT INTO report_schedules (name, frequency, recipients, filters, send_hour, day_of_week, is_active, next_run_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetReportSchedule :one
SELECT * FROM report_schedules WHERE id = $1;

-- name: ListReportSchedules :many
SELECT * FROM report_schedules ORDER BY created_at DESC;

-- name: UpdateReportSchedule :one
UPDATE report_schedules
SET name        = $2,
    frequency   = $3,
    recipients  = $4,
    filters     = $5,
    send_hour   = $6,
    day_of_week = $7,
    is_active   = $8,
    next_run_at = $9,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteReportSchedule :exec
DELETE FROM report_schedules WHERE id = $1;

-- name: GetDueSchedules :many
SELECT * FROM report_schedules
WHERE is_active = true AND next_run_at <= NOW()
ORDER BY next_run_at ASC;

-- name: MarkScheduleSent :one
UPDATE report_schedules
SET last_sent_at = NOW(),
    next_run_at  = $2,
    updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: CountNewCasesInPeriod :one
SELECT COUNT(*) FROM cases
WHERE deleted_at IS NULL
  AND created_at >= $1;
