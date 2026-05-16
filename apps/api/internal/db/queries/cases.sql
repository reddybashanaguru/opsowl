-- name: CreateCase :one
INSERT INTO cases (
  case_number, title, description, status, priority,
  assigned_to, created_by, organization_id, due_date, sla_breach_at, tags
) VALUES (
  FORMAT('OWL-%s-%05d', EXTRACT(YEAR FROM NOW())::TEXT, nextval('case_number_seq')),
  @title, @description, @status, @priority,
  @assigned_to, @created_by, @organization_id, @due_date, @sla_breach_at, @tags
)
RETURNING *;

-- name: GetCaseByID :one
SELECT
  c.*,
  creator.full_name  AS created_by_name,
  creator.email      AS created_by_email,
  assignee.full_name AS assigned_to_name,
  assignee.email     AS assigned_to_email,
  assignee.role      AS assigned_to_role
FROM cases c
JOIN users creator  ON creator.id  = c.created_by
LEFT JOIN users assignee ON assignee.id = c.assigned_to
WHERE c.id = $1 AND c.deleted_at IS NULL;

-- name: ListCases :many
SELECT
  c.*,
  creator.full_name  AS created_by_name,
  assignee.full_name AS assigned_to_name
FROM cases c
JOIN users creator ON creator.id = c.created_by
LEFT JOIN users assignee ON assignee.id = c.assigned_to
WHERE c.deleted_at IS NULL
  AND c.organization_id = @organization_id
  AND (sqlc.narg('status')::case_status IS NULL OR c.status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::case_priority IS NULL OR c.priority = sqlc.narg('priority'))
  AND (sqlc.narg('assigned_to')::uuid IS NULL OR c.assigned_to = sqlc.narg('assigned_to'))
  AND (sqlc.narg('sla_breached')::boolean IS NULL OR c.sla_breached = sqlc.narg('sla_breached'))
  AND (sqlc.narg('from_time')::timestamptz IS NULL OR c.created_at >= sqlc.narg('from_time'))
  AND (sqlc.narg('to_time')::timestamptz IS NULL OR c.created_at <= sqlc.narg('to_time'))
  AND (sqlc.narg('cursor')::uuid IS NULL OR c.id < sqlc.narg('cursor'))
ORDER BY c.created_at DESC
LIMIT @row_limit;

-- name: CountCases :one
SELECT COUNT(*) FROM cases
WHERE deleted_at IS NULL
  AND organization_id = @organization_id
  AND (sqlc.narg('status')::case_status IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::case_priority IS NULL OR priority = sqlc.narg('priority'))
  AND (sqlc.narg('assigned_to')::uuid IS NULL OR assigned_to = sqlc.narg('assigned_to'))
  AND (sqlc.narg('sla_breached')::boolean IS NULL OR sla_breached = sqlc.narg('sla_breached'));

-- name: UpdateCase :one
UPDATE cases SET
  title       = COALESCE(sqlc.narg(title), title),
  description = COALESCE(sqlc.narg(description), description),
  priority    = COALESCE(sqlc.narg(priority), priority),
  due_date    = COALESCE(sqlc.narg(due_date), due_date),
  updated_at  = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCaseStatus :one
UPDATE cases SET
  status        = @status,
  sla_breach_at = COALESCE(sqlc.narg(sla_breach_at), sla_breach_at),
  sla_breached  = COALESCE(sqlc.narg(sla_breached), sla_breached),
  updated_at    = NOW()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCaseAssignment :one
UPDATE cases SET
  assigned_to = @assigned_to,
  updated_at  = NOW()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCaseTags :one
UPDATE cases SET
  tags       = @tags,
  updated_at = NOW()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCase :one
UPDATE cases SET
  deleted_at = NOW(),
  updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CreateCaseActivity :one
INSERT INTO case_activities (case_id, actor_id, event_type, payload)
VALUES (@case_id, @actor_id, @event_type, @payload)
RETURNING *;

-- name: ListCaseActivities :many
SELECT
  a.*,
  u.full_name AS actor_name,
  u.email     AS actor_email
FROM case_activities a
JOIN users u ON u.id = a.actor_id
WHERE a.case_id = @case_id
  AND (sqlc.narg('cursor')::uuid IS NULL OR a.id < sqlc.narg('cursor'))
ORDER BY a.created_at DESC
LIMIT @row_limit;

-- name: CreateCaseAssignment :one
INSERT INTO case_assignments (case_id, assigned_to, assigned_by)
VALUES (@case_id, @assigned_to, @assigned_by)
RETURNING *;

-- name: CloseLastAssignment :exec
UPDATE case_assignments SET
  unassigned_at = NOW()
WHERE case_id = $1 AND unassigned_at IS NULL;

-- name: UpsertCaseTag :one
INSERT INTO case_tags (name) VALUES ($1)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: AddCaseWatcher :exec
INSERT INTO case_watchers (case_id, user_id) VALUES (@case_id, @user_id)
ON CONFLICT DO NOTHING;

-- name: GetCaseWatchers :many
SELECT u.id, u.full_name, u.email
FROM case_watchers cw
JOIN users u ON u.id = cw.user_id
WHERE cw.case_id = $1;
