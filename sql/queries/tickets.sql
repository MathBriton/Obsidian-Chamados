-- name: CreateTicket :one
INSERT INTO tickets (
    tenant_id, title, description, status, priority, category_id, created_by,
    first_response_due_at, resolution_due_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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
  AND (sqlc.narg('team_id') IS NULL OR assigned_team_id = sqlc.narg('team_id'))
  AND (sqlc.narg('search') IS NULL
       OR title LIKE '%' || sqlc.narg('search') || '%'
       OR description LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('breached_before') IS NULL OR (
        resolved_at IS NULL AND closed_at IS NULL AND (
            (resolution_due_at IS NOT NULL
             AND resolution_due_at < sqlc.narg('breached_before'))
         OR (first_response_due_at IS NOT NULL AND first_responded_at IS NULL
             AND first_response_due_at < sqlc.narg('breached_before')))))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: ListTicketsByCreator :many
SELECT * FROM tickets
WHERE tenant_id = sqlc.arg('tenant_id') AND created_by = sqlc.arg('created_by')
  AND (sqlc.narg('status') IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority') IS NULL OR priority = sqlc.narg('priority'))
  AND (sqlc.narg('assigned_to') IS NULL OR assigned_to = sqlc.narg('assigned_to'))
  AND (sqlc.narg('team_id') IS NULL OR assigned_team_id = sqlc.narg('team_id'))
  AND (sqlc.narg('search') IS NULL
       OR title LIKE '%' || sqlc.narg('search') || '%'
       OR description LIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('breached_before') IS NULL OR (
        resolved_at IS NULL AND closed_at IS NULL AND (
            (resolution_due_at IS NOT NULL
             AND resolution_due_at < sqlc.narg('breached_before'))
         OR (first_response_due_at IS NOT NULL AND first_responded_at IS NULL
             AND first_response_due_at < sqlc.narg('breached_before')))))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountTicketsByStatus :many
SELECT status, COUNT(*) AS total FROM tickets
WHERE tenant_id = sqlc.arg('tenant_id')
  AND (sqlc.narg('created_by') IS NULL OR created_by = sqlc.narg('created_by'))
GROUP BY status;

-- name: CountTicketsByPriority :many
SELECT priority, COUNT(*) AS total FROM tickets
WHERE tenant_id = sqlc.arg('tenant_id')
  AND (sqlc.narg('created_by') IS NULL OR created_by = sqlc.narg('created_by'))
GROUP BY priority;

-- name: CountUnassignedActiveTickets :one
SELECT COUNT(*) FROM tickets
WHERE tenant_id = sqlc.arg('tenant_id')
  AND (sqlc.narg('created_by') IS NULL OR created_by = sqlc.narg('created_by'))
  AND assigned_to IS NULL
  AND status NOT IN ('resolved', 'closed');

-- name: UpdateTicket :one
UPDATE tickets
SET title                 = ?,
    description           = ?,
    status                = ?,
    priority              = ?,
    category_id           = ?,
    assigned_to           = ?,
    assigned_team_id      = ?,
    resolved_at           = ?,
    closed_at             = ?,
    first_response_due_at = ?,
    resolution_due_at     = ?,
    first_responded_at    = ?,
    updated_at            = CURRENT_TIMESTAMP
WHERE tenant_id = ? AND id = ?
RETURNING *;

-- name: StampFirstResponse :exec
UPDATE tickets
SET first_responded_at = ?
WHERE tenant_id = ? AND id = ? AND first_responded_at IS NULL;
