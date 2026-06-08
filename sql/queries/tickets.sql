-- name: CreateTicket :one
INSERT INTO tickets (tenant_id, title, description, status, priority, category_id, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTicketByID :one
SELECT * FROM tickets
WHERE tenant_id = ? AND id = ?
LIMIT 1;

-- name: ListTicketsByTenant :many
SELECT * FROM tickets
WHERE tenant_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListTicketsByCreator :many
SELECT * FROM tickets
WHERE tenant_id = ? AND created_by = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateTicket :one
UPDATE tickets
SET title            = ?,
    description      = ?,
    status           = ?,
    priority         = ?,
    category_id      = ?,
    assigned_to      = ?,
    assigned_team_id = ?,
    resolved_at      = ?,
    closed_at        = ?,
    updated_at       = CURRENT_TIMESTAMP
WHERE tenant_id = ? AND id = ?
RETURNING *;
