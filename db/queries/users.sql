-- name: GetUserByID :one
SELECT id, name, username, password_hash, role, is_active, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, name, username, password_hash, role, is_active, created_at, updated_at
FROM users
WHERE username = $1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (name, username, password_hash, role, is_active)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, username, password_hash, role, is_active, created_at, updated_at;

-- name: ListUsers :many
SELECT id, name, username, role, is_active, created_at, updated_at
FROM users
ORDER BY id ASC;

-- name: UpdateUser :one
UPDATE users
SET name = COALESCE(sqlc.narg('name'), name),
    role = COALESCE(sqlc.narg('role'), role),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING id, name, username, role, is_active, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, revoked, created_at;

-- name: GetRefreshToken :one
SELECT id, user_id, token_hash, expires_at, revoked, created_at
FROM refresh_tokens
WHERE token_hash = $1 AND revoked = false AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = true
WHERE token_hash = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = true
WHERE user_id = $1;
