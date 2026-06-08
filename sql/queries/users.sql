-- name: CreateUser :one
INSERT INTO users (tenant_id, name, email, password_hash, role)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE tenant_id = ? AND email = ?
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE tenant_id = ? AND id = ?
LIMIT 1;

-- name: ListUsersByTenant :many
SELECT * FROM users
WHERE tenant_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = 0, updated_at = CURRENT_TIMESTAMP
WHERE tenant_id = ? AND id = ?;
