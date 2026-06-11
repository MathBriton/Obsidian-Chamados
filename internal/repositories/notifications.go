package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// NotificationRepository dá acesso à tabela notifications, escopada por
// tenant_id + user_id — cada usuário só enxerga as próprias (ADR-001).
type NotificationRepository struct {
	q db.Querier
}

// CreateNotificationInput agrega os dados de uma notificação in-app.
type CreateNotificationInput struct {
	TenantID int64
	UserID   int64
	TicketID int64
	Kind     string
	Message  string
}

// Create registra uma notificação para um destinatário.
func (r *NotificationRepository) Create(ctx context.Context, in CreateNotificationInput) (db.Notification, error) {
	return r.q.CreateNotification(ctx, db.CreateNotificationParams{
		TenantID: in.TenantID,
		UserID:   in.UserID,
		TicketID: in.TicketID,
		Kind:     in.Kind,
		Message:  in.Message,
	})
}

// ListByUser lista as notificações do usuário no tenant, da mais recente para a
// mais antiga. Com unreadOnly, retorna apenas as não-lidas.
func (r *NotificationRepository) ListByUser(ctx context.Context, tenantID, userID int64, unreadOnly bool, limit, offset int64) ([]db.Notification, error) {
	var unread interface{}
	if unreadOnly {
		unread = true
	}
	return r.q.ListNotificationsByUser(ctx, db.ListNotificationsByUserParams{
		TenantID:   tenantID,
		UserID:     userID,
		UnreadOnly: unread,
		Limit:      limit,
		Offset:     offset,
	})
}

// CountUnread devolve quantas notificações não-lidas o usuário tem (badge).
func (r *NotificationRepository) CountUnread(ctx context.Context, tenantID, userID int64) (int64, error) {
	return r.q.CountUnreadNotifications(ctx, db.CountUnreadNotificationsParams{TenantID: tenantID, UserID: userID})
}

// GetByID busca uma notificação do usuário pelo id. Ausência (ou de outro
// usuário/tenant) retorna models.ErrNotFound (ADR-003).
func (r *NotificationRepository) GetByID(ctx context.Context, tenantID, userID, id int64) (db.Notification, error) {
	n, err := r.q.GetNotificationByID(ctx, db.GetNotificationByIDParams{TenantID: tenantID, UserID: userID, ID: id})
	if err != nil {
		return db.Notification{}, notFound(err)
	}
	return n, nil
}

// MarkRead marca uma notificação como lida (idempotente: re-marcar não move o
// read_at, pois o UPDATE só toca linhas ainda não-lidas).
func (r *NotificationRepository) MarkRead(ctx context.Context, tenantID, userID, id int64, at time.Time) error {
	return r.q.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		ReadAt:   sql.NullTime{Time: at, Valid: true},
		TenantID: tenantID,
		UserID:   userID,
		ID:       id,
	})
}

// MarkAllRead marca todas as não-lidas do usuário como lidas.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, tenantID, userID int64, at time.Time) error {
	return r.q.MarkAllNotificationsRead(ctx, db.MarkAllNotificationsReadParams{
		ReadAt:   sql.NullTime{Time: at, Valid: true},
		TenantID: tenantID,
		UserID:   userID,
	})
}
