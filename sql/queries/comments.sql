-- name: CreateComment :one
INSERT INTO comments (tenant_id, ticket_id, author_id, body, is_internal)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListCommentsByTicket :many
SELECT c.*, u.name AS author_name
FROM comments c
JOIN users u ON u.id = c.author_id
WHERE c.tenant_id = ? AND c.ticket_id = ?
ORDER BY c.created_at ASC;

-- name: ListPublicCommentsByTicket :many
SELECT c.*, u.name AS author_name
FROM comments c
JOIN users u ON u.id = c.author_id
WHERE c.tenant_id = ? AND c.ticket_id = ? AND c.is_internal = 0
ORDER BY c.created_at ASC;
