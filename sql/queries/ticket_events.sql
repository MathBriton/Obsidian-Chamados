-- name: CreateTicketEvent :one
INSERT INTO ticket_events (tenant_id, ticket_id, actor_id, kind, old_value, new_value)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListTicketEvents :many
SELECT * FROM ticket_events
WHERE tenant_id = ? AND ticket_id = ?
ORDER BY created_at, id;
