-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, role)
VALUES ($1, $2, $3, $4)
RETURNING id, email, full_name, role, is_active, last_login_at, created_at, updated_at, deleted_at;

-- name: ListUsers :many
SELECT id, email, full_name, role, is_active, last_login_at, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('role')::user_role IS NULL OR role = sqlc.narg('role')::user_role)
  AND (sqlc.narg('cursor')::uuid IS NULL OR (created_at, id) < (
        SELECT created_at, id FROM users WHERE id = sqlc.narg('cursor')::uuid
      ))
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE deleted_at IS NULL
  AND (sqlc.narg('role')::user_role IS NULL OR role = sqlc.narg('role')::user_role);

-- name: UpdateUser :one
UPDATE users
SET full_name  = COALESCE(sqlc.narg('full_name'), full_name),
    role       = COALESCE(sqlc.narg('role')::user_role, role),
    is_active  = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, email, full_name, role, is_active, last_login_at, created_at, updated_at, deleted_at;

-- name: SoftDeleteUser :one
UPDATE users
SET deleted_at = NOW(), updated_at = NOW(), is_active = FALSE
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, deleted_at;
