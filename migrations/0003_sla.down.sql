-- 0003_sla.down.sql — reverte o módulo SLA & Prazos.

DROP INDEX IF EXISTS idx_tickets_resolution_due;

ALTER TABLE tickets DROP COLUMN first_responded_at;
ALTER TABLE tickets DROP COLUMN resolution_due_at;
ALTER TABLE tickets DROP COLUMN first_response_due_at;

DROP TABLE IF EXISTS sla_policies;
