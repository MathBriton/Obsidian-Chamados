-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (tenant_id, user_id, token_hash, expires_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = ?
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE token_hash = ?;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE tenant_id = ? AND user_id = ? AND revoked_at IS NULL;
