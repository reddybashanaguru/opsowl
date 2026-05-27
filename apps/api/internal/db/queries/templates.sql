-- ── Case Templates ─────────────────────────────────────────────────────────────

-- name: InsertCaseTemplate :one
INSERT INTO case_templates (name, description_skeleton, priority, tags, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCaseTemplate :one
SELECT * FROM case_templates WHERE id = $1;

-- name: ListCaseTemplates :many
SELECT * FROM case_templates
WHERE ($1::boolean IS NULL OR is_active = $1)
ORDER BY name ASC;

-- name: UpdateCaseTemplate :one
UPDATE case_templates
SET name                 = $2,
    description_skeleton = $3,
    priority             = $4,
    tags                 = $5,
    is_active            = $6,
    updated_at           = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteCaseTemplate :exec
DELETE FROM case_templates WHERE id = $1;

-- ── Template tasks ─────────────────────────────────────────────────────────────

-- name: InsertTemplateTask :one
INSERT INTO case_template_tasks (template_id, title, description, priority, sort_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListTemplateTasks :many
SELECT * FROM case_template_tasks
WHERE template_id = $1
ORDER BY sort_order ASC, id ASC;

-- name: DeleteTemplateTasksByTemplate :exec
DELETE FROM case_template_tasks WHERE template_id = $1;
