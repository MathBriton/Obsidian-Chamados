-- name: CreateTenant :one
INSERT INTO tenants (name, slug)
VALUES (?, ?)
RETURNING *;

-- name: GetTenantBySlug :one
SELECT * FROM tenants
WHERE slug = ?
LIMIT 1;

-- name: GetTenantByID :one
SELECT * FROM tenants
WHERE id = ?
LIMIT 1;
