-- 0004_notifications.down.sql — reverte o módulo de notificações.

DROP INDEX IF EXISTS idx_notifications_user;
DROP TABLE IF EXISTS notifications;
