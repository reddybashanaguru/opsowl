-- name: GetUserByEmail :one
SELECT id, email, password_hash, full_name, role, is_active, last_login_at,
       created_at, updated_at, deleted_at
FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, password_hash, full_name, role, is_active, last_login_at,
       created_at, updated_at, deleted_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

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
INSERT INTO audit_logs (event_code, actor_id, ip_address, resource_type, resource_id, metadata, severity)
VALUES ($1, $2, $3, $4, $5, $6, $7);
