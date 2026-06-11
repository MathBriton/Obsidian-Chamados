-- name: UpsertSLAPolicy :one
INSERT INTO sla_policies (tenant_id, priority, first_response_mins, resolution_mins)
VALUES (?, ?, ?, ?)
ON CONFLICT (tenant_id, priority)
DO UPDATE SET first_response_mins = excluded.first_response_mins,
              resolution_mins     = excluded.resolution_mins,
              updated_at          = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetSLAPolicyByPriority :one
SELECT * FROM sla_policies
WHERE tenant_id = ? AND priority = ?
LIMIT 1;

-- name: ListSLAPoliciesByTenant :many
SELECT * FROM sla_policies
WHERE tenant_id = ?
ORDER BY CASE priority
    WHEN 'critical' THEN 0
    WHEN 'high'     THEN 1
    WHEN 'medium'   THEN 2
    WHEN 'low'      THEN 3
END;
