-- name: CreateTask :one
INSERT INTO tasks (case_id, created_by, assigned_to, timeline_event_id, title, description, priority, due_date, is_ai_suggested, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
    COALESCE((SELECT MAX(sort_order) + 1 FROM tasks WHERE case_id = $1 AND deleted_at IS NULL), 0))
RETURNING *;

-- name: GetTaskByID :one
SELECT
    t.*,
    u_creator.full_name AS created_by_name,
    u_creator.email     AS created_by_email,
    u_assign.full_name  AS assigned_to_name,
    u_assign.email      AS assigned_to_email,
    u_assign.role       AS assigned_to_role
FROM tasks t
JOIN users u_creator ON u_creator.id = t.created_by
LEFT JOIN users u_assign ON u_assign.id = t.assigned_to
WHERE t.id = $1 AND t.deleted_at IS NULL;

-- name: ListTasksByCase :many
SELECT
    t.*,
    u_creator.full_name AS created_by_name,
    u_assign.full_name  AS assigned_to_name,
    u_assign.email      AS assigned_to_email
FROM tasks t
JOIN users u_creator ON u_creator.id = t.created_by
LEFT JOIN users u_assign ON u_assign.id = t.assigned_to
WHERE t.case_id = $1
  AND t.deleted_at IS NULL
  AND (sqlc.narg('status')::task_status IS NULL OR t.status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::task_priority IS NULL OR t.priority = sqlc.narg('priority'))
  AND (sqlc.narg('assigned_to')::uuid IS NULL OR t.assigned_to = sqlc.narg('assigned_to'))
  AND (sqlc.narg('overdue_only')::boolean IS NULL OR sqlc.narg('overdue_only') = FALSE OR t.is_overdue = TRUE)
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR t.id < sqlc.narg('cursor_id'))
ORDER BY t.sort_order ASC, t.created_at ASC
LIMIT @row_limit;

-- name: ListMyTasks :many
SELECT
    t.*,
    c.case_number,
    c.title AS case_title,
    u_creator.full_name AS created_by_name,
    u_assign.full_name  AS assigned_to_name,
    u_assign.email      AS assigned_to_email
FROM tasks t
JOIN cases c ON c.id = t.case_id AND c.deleted_at IS NULL
JOIN users u_creator ON u_creator.id = t.created_by
LEFT JOIN users u_assign ON u_assign.id = t.assigned_to
WHERE t.assigned_to = $1
  AND t.deleted_at IS NULL
  AND (sqlc.narg('status')::task_status IS NULL OR t.status = sqlc.narg('status'))
  AND (sqlc.narg('overdue_only')::boolean IS NULL OR sqlc.narg('overdue_only') = FALSE OR t.is_overdue = TRUE)
ORDER BY
    t.is_overdue DESC,
    t.due_date ASC NULLS LAST,
    t.priority ASC
LIMIT @row_limit;

-- name: CountTasksByCase :one
SELECT COUNT(*) FROM tasks
WHERE case_id = $1 AND deleted_at IS NULL
  AND (sqlc.narg('status')::task_status IS NULL OR status = sqlc.narg('status'));

-- name: UpdateTask :one
UPDATE tasks SET
    title           = COALESCE(sqlc.narg('title')::text,        title),
    description     = COALESCE(sqlc.narg('description')::text,  description),
    priority        = COALESCE(sqlc.narg('priority')::task_priority, priority),
    due_date        = CASE WHEN sqlc.narg('clear_due_date')::boolean THEN NULL
                           ELSE COALESCE(sqlc.narg('due_date')::timestamptz, due_date) END,
    assigned_to     = CASE WHEN sqlc.narg('clear_assigned_to')::boolean THEN NULL
                           ELSE COALESCE(sqlc.narg('assigned_to')::uuid, assigned_to) END,
    status          = COALESCE(sqlc.narg('status')::task_status, status),
    completed_at    = sqlc.narg('completed_at')::timestamptz,
    is_overdue      = COALESCE(sqlc.narg('is_overdue')::boolean, is_overdue),
    sort_order      = COALESCE(sqlc.narg('sort_order')::integer,  sort_order)
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CompleteTask :one
UPDATE tasks SET
    status       = 'done',
    completed_at = NOW(),
    is_overdue   = FALSE
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTask :one
UPDATE tasks SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id;

-- name: CreateTaskComment :one
INSERT INTO task_comments (task_id, author_id, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTaskComments :many
SELECT
    tc.*,
    u.full_name AS author_name,
    u.email     AS author_email
FROM task_comments tc
JOIN users u ON u.id = tc.author_id
WHERE tc.task_id = $1 AND tc.deleted_at IS NULL
  AND (sqlc.narg('cursor_id')::uuid IS NULL OR tc.id < sqlc.narg('cursor_id'))
ORDER BY tc.created_at DESC
LIMIT @row_limit;

-- name: CountTaskComments :one
SELECT COUNT(*) FROM task_comments WHERE task_id = $1 AND deleted_at IS NULL;

-- name: FlagOverdueTasks :many
UPDATE tasks SET is_overdue = TRUE
WHERE due_date < NOW()
  AND status NOT IN ('done', 'cancelled')
  AND is_overdue = FALSE
  AND deleted_at IS NULL
RETURNING id, case_id, assigned_to, title;
