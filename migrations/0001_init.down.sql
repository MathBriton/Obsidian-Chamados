-- 0001_init.down.sql
-- Reverte o schema inicial. Ordem inversa por causa das foreign keys.

DROP INDEX IF EXISTS idx_refresh_token_hash;
DROP INDEX IF EXISTS idx_comments_ticket;
DROP INDEX IF EXISTS idx_tickets_created_by;
DROP INDEX IF EXISTS idx_tickets_assigned;
DROP INDEX IF EXISTS idx_tickets_tenant_status;
DROP INDEX IF EXISTS idx_categories_tenant;
DROP INDEX IF EXISTS idx_teams_tenant;
DROP INDEX IF EXISTS idx_users_tenant;

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
