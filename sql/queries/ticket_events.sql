-- name: CreateTicketEvent :one
INSERT INTO ticket_events (tenant_id, ticket_id, actor_id, kind, old_value, new_value)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTicketEvents :many
SELECT e.*, u.name AS actor_name
FROM ticket_events e
JOIN users u ON u.id = e.actor_id
WHERE e.tenant_id = ? AND e.ticket_id = ?
ORDER BY e.created_at, e.id;
