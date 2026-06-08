-- name: CreateCategory :one
INSERT INTO categories (tenant_id, name)
VALUES (?, ?)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories
WHERE tenant_id = ? AND id = ?
LIMIT 1;

-- name: ListCategoriesByTenant :many
SELECT * FROM categories
WHERE tenant_id = ?
ORDER BY name;
