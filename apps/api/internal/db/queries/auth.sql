-- name: GetUserByEmail :one
SELECT id, email, password_hash, full_name, role, is_active, last_login_at,
       created_at, updated_at, deleted_at,
       failed_login_attempts, locked_until, lockout_count
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, password_hash, full_name, role, is_active, last_login_at,
       created_at, updated_at, deleted_at,
       failed_login_attempts, locked_until, lockout_count
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: IncrementFailedLoginAttempts :exec
UPDATE users SET failed_login_attempts = failed_login_attempts + 1, updated_at = NOW()
WHERE id = $1;

-- name: LockAccount :exec
UPDATE users
SET locked_until = $2, lockout_count = lockout_count + 1,
    failed_login_attempts = 0, updated_at = NOW()
WHERE id = $1;

-- name: ResetLoginAttempts :exec
UPDATE users SET failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
WHERE id = $1;

-- name: AdminUnlockAccount :exec
UPDATE users SET failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
WHERE id = $1;

-- name: AddPasswordHistory :exec
INSERT INTO password_history (user_id, password_hash) VALUES ($1, $2);

-- name: GetPasswordHistory :many
SELECT id, user_id, password_hash, created_at
FROM password_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: PrunePasswordHistory :exec
DELETE FROM password_history
WHERE id IN (
  SELECT id FROM password_history
  WHERE user_id = $1
  ORDER BY created_at DESC
  OFFSET $2
);

-- name: UpdatePasswordChangedAt :exec
UPDATE users SET password_changed_at = NOW(),
  password_expires_at = CASE WHEN $2 > 0 THEN NOW() + ($2 * INTERVAL '1 day') ELSE NULL END,
  updated_at = NOW()
WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, revoked_at, created_at;

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
FROM refresh_tokens
WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreateSession :one
INSERT INTO sessions (user_id, ip_address, user_agent)
VALUES ($1, $2, $3)
RETURNING id, user_id, ip_address, user_agent, created_at, last_active_at;

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (event_code, actor_id, ip_address, resource_type, resource_id, metadata, severity,
                        seq_num, prev_hash, entry_hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetLastAuditEntry :one
SELECT COALESCE(MAX(seq_num), 0) AS seq_num, COALESCE(
  (SELECT entry_hash FROM audit_logs WHERE seq_num = MAX(al.seq_num)),
  'genesis'
) AS entry_hash
FROM audit_logs al
WHERE seq_num IS NOT NULL;

-- name: ListAuditChain :many
SELECT id, seq_num, event_code, actor_id, ip_address, metadata, severity, prev_hash, entry_hash, created_at
FROM audit_logs
WHERE seq_num IS NOT NULL
  AND seq_num > sqlc.arg(from_seq)
ORDER BY seq_num ASC
LIMIT sqlc.arg(row_limit);

-- name: ListAuditLogs :many
SELECT
  a.id,
  a.event_code,
  a.actor_id,
  u.full_name AS actor_name,
  u.email     AS actor_email,
  a.ip_address,
  a.resource_type,
  a.resource_id,
  a.metadata,
  a.severity,
  a.created_at
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE (sqlc.narg('severity')::text IS NULL OR a.severity = sqlc.narg('severity'))
  AND (sqlc.narg('event_code')::text IS NULL OR a.event_code ILIKE '%' || sqlc.narg('event_code')::text || '%')
  AND (sqlc.narg('actor_id')::uuid IS NULL OR a.actor_id = sqlc.narg('actor_id'))
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR a.created_at < sqlc.narg('cursor_time')::timestamptz
    OR (a.created_at = sqlc.narg('cursor_time')::timestamptz AND a.id < sqlc.narg('cursor_id')::uuid)
  )
ORDER BY a.created_at DESC, a.id DESC
LIMIT @row_limit;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs a
WHERE (sqlc.narg('severity')::text IS NULL OR a.severity = sqlc.narg('severity'))
  AND (sqlc.narg('event_code')::text IS NULL OR a.event_code ILIKE '%' || sqlc.narg('event_code')::text || '%')
  AND (sqlc.narg('actor_id')::uuid IS NULL OR a.actor_id = sqlc.narg('actor_id'));
