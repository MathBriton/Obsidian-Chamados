-- name: CreateComment :one
INSERT INTO comments (tenant_id, ticket_id, author_id, body, is_internal)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListCommentsByTicket :many
SELECT * FROM comments
WHERE tenant_id = ? AND ticket_id = ?
ORDER BY created_at ASC;

-- name: ListPublicCommentsByTicket :many
SELECT * FROM comments
WHERE tenant_id = ? AND ticket_id = ? AND is_internal = 0
ORDER BY created_at ASC;
