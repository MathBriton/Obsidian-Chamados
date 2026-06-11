-- 0002_ticket_events.up.sql
-- Histórico (auditoria) de tickets: registra abertura e mudanças relevantes
-- (status, prioridade, categoria, responsável, equipe). Valores são snapshots
-- legíveis no momento do evento (enum ou nome), pois nomes podem mudar depois.

CREATE TABLE ticket_events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id  INTEGER  NOT NULL REFERENCES tenants(id),
    ticket_id  INTEGER  NOT NULL REFERENCES tickets(id),
    actor_id   INTEGER  NOT NULL REFERENCES users(id),
    kind       TEXT     NOT NULL CHECK (kind IN (
                   'created', 'status_changed', 'priority_changed',
                   'category_changed', 'assignee_changed', 'team_changed')),
    old_value  TEXT,
    new_value  TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ticket_events_ticket ON ticket_events (ticket_id, created_at);
