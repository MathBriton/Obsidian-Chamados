-- 0004_notifications.up.sql
-- Notificações in-app: geradas event-driven na mesma transação do evento do
-- ticket (sem worker/cron). No MVP, apenas comentários públicos geram
-- notificação; o enum de kind já contempla eventos futuros sem nova migration.

CREATE TABLE notifications (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id  INTEGER  NOT NULL REFERENCES tenants(id),
    user_id    INTEGER  NOT NULL REFERENCES users(id),
    ticket_id  INTEGER  NOT NULL REFERENCES tickets(id),
    kind       TEXT     NOT NULL CHECK (kind IN ('comment', 'assigned', 'status_changed')),
    message    TEXT     NOT NULL,
    read_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Cobre a listagem do usuário, o filtro de não-lidas e o contador do badge.
CREATE INDEX idx_notifications_user ON notifications (tenant_id, user_id, read_at, created_at);
