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
WHERE tenant_id = sqlc.arg('tenant_id')
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority') IS NULL OR priority = sqlc.narg('priority'))
  AND (sqlc.narg('assigned_to') IS NULL OR assigned_to = sqlc.narg('assigned_to'))
  AND (sqlc.narg('search') IS NULL
       OR title LIKE '%' || sqlc.narg('search') || '%'
       OR description LIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListTicketsByCreator :many
SELECT * FROM tickets
WHERE tenant_id = sqlc.arg('tenant_id') AND created_by = sqlc.arg('created_by')
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority') IS NULL OR priority = sqlc.narg('priority'))
  AND (sqlc.narg('assigned_to') IS NULL OR assigned_to = sqlc.narg('assigned_to'))
  AND (sqlc.narg('search') IS NULL
       OR title LIKE '%' || sqlc.narg('search') || '%'
       OR description LIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

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
