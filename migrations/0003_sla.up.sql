-- 0003_sla.up.sql
-- Módulo SLA & Prazos: políticas de SLA por prioridade e prazos por ticket.
-- O estado do SLA (ok/at_risk/breached/met) é DERIVADO em tempo de leitura a
-- partir dos timestamps + "agora", nunca armazenado (sem job de varredura).
-- Relógio em tempo absoluto corrido — sem pausa em waiting_customer (MVP).

CREATE TABLE sla_policies (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id           INTEGER  NOT NULL REFERENCES tenants(id),
    priority            TEXT     NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    first_response_mins INTEGER  NOT NULL CHECK (first_response_mins > 0),
    resolution_mins     INTEGER  NOT NULL CHECK (resolution_mins > 0),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, priority)
);

-- Prazos do ticket. Nullable: sem política para a prioridade => sem SLA.
-- Os valores são gravados como time.Time (mesmo formato de resolved_at), o que
-- mantém a comparação do filtro ?sla=breached consistente.
ALTER TABLE tickets ADD COLUMN first_response_due_at DATETIME;
ALTER TABLE tickets ADD COLUMN resolution_due_at     DATETIME;
ALTER TABLE tickets ADD COLUMN first_responded_at    DATETIME;

CREATE INDEX idx_tickets_resolution_due ON tickets (tenant_id, resolution_due_at);

-- Backfill de políticas padrão para os tenants já existentes. Os valores
-- canônicos vivem em services.DefaultSLAPolicies (aplicados no registro de
-- novos tenants); manter ambos em sincronia. Minutos: 1ª resposta / resolução.
INSERT INTO sla_policies (tenant_id, priority, first_response_mins, resolution_mins)
SELECT id, 'critical', 15, 120   FROM tenants
UNION ALL SELECT id, 'high',   30, 240   FROM tenants
UNION ALL SELECT id, 'medium', 60, 480   FROM tenants
UNION ALL SELECT id, 'low',    120, 1440 FROM tenants;
