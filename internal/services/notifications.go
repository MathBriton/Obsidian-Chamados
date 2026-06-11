package services

import (
	"context"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// Tipos de notificação. No MVP só "comment" é emitido; os demais já existem no
// schema para evolução futura sem nova migration.
const (
	notifyKindComment = "comment"
)

// NotificationService gere as notificações in-app de um usuário. Cada usuário
// só acessa as próprias (escopo tenant_id + user_id).
type NotificationService struct {
	store *repositories.Store
}

// NewNotificationService monta o serviço de notificações.
func NewNotificationService(store *repositories.Store) *NotificationService {
	return &NotificationService{store: store}
}

// List devolve as notificações do actor, da mais recente para a mais antiga.
// Com unreadOnly, retorna apenas as não-lidas. A paginação é normalizada como
// na listagem de tickets.
func (s *NotificationService) List(ctx context.Context, actor Actor, unreadOnly bool, limit, offset int64) ([]db.Notification, error) {
	if limit <= 0 || limit > defaultListLimit {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.Notifications.ListByUser(ctx, actor.TenantID, actor.UserID, unreadOnly, limit, offset)
}

// UnreadCount devolve a contagem de não-lidas do actor (badge).
func (s *NotificationService) UnreadCount(ctx context.Context, actor Actor) (int64, error) {
	return s.store.Notifications.CountUnread(ctx, actor.TenantID, actor.UserID)
}

// MarkRead marca uma notificação do actor como lida. Notificação inexistente ou
// de outro usuário/tenant responde ErrNotFound (ADR-003). Idempotente.
func (s *NotificationService) MarkRead(ctx context.Context, actor Actor, id int64) error {
	if _, err := s.store.Notifications.GetByID(ctx, actor.TenantID, actor.UserID, id); err != nil {
		return err
	}
	return s.store.Notifications.MarkRead(ctx, actor.TenantID, actor.UserID, id, time.Now().UTC())
}

// MarkAllRead marca todas as não-lidas do actor como lidas.
func (s *NotificationService) MarkAllRead(ctx context.Context, actor Actor) error {
	return s.store.Notifications.MarkAllRead(ctx, actor.TenantID, actor.UserID, time.Now().UTC())
}
