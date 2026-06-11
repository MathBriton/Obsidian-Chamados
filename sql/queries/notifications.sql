-- name: CreateNotification :one
INSERT INTO notifications (tenant_id, user_id, ticket_id, kind, message)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListNotificationsByUser :many
SELECT * FROM notifications
WHERE tenant_id = sqlc.arg('tenant_id') AND user_id = sqlc.arg('user_id')
  AND (sqlc.narg('unread_only') IS NULL OR read_at IS NULL)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE tenant_id = ? AND user_id = ? AND read_at IS NULL;

-- name: GetNotificationByID :one
SELECT * FROM notifications
WHERE tenant_id = ? AND user_id = ? AND id = ?
LIMIT 1;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET read_at = ?
WHERE tenant_id = ? AND user_id = ? AND id = ? AND read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET read_at = ?
WHERE tenant_id = ? AND user_id = ? AND read_at IS NULL;
