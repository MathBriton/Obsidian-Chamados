-- name: CreateTeam :one
INSERT INTO teams (tenant_id, name)
VALUES (?, ?)
RETURNING *;

-- name: GetTeamByID :one
SELECT * FROM teams
WHERE tenant_id = ? AND id = ?
LIMIT 1;

-- name: ListTeamsByTenant :many
SELECT * FROM teams
WHERE tenant_id = ?
ORDER BY name;

-- name: AddTeamMember :exec
INSERT OR IGNORE INTO team_members (team_id, user_id, tenant_id)
VALUES (?, ?, ?);

-- name: RemoveTeamMember :exec
DELETE FROM team_members
WHERE tenant_id = ? AND team_id = ? AND user_id = ?;

-- name: ListTeamMembersByTenant :many
SELECT tm.team_id, u.id, u.name, u.role
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.tenant_id = ? AND u.is_active = 1
ORDER BY u.name;
