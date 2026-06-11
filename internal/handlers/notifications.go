package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

type notificationResponse struct {
	ID        int64      `json:"id"`
	TicketID  int64      `json:"ticket_id"`
	Kind      string     `json:"kind"`
	Message   string     `json:"message"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func toNotificationResponse(n db.Notification) notificationResponse {
	return notificationResponse{
		ID:        n.ID,
		TicketID:  n.TicketID,
		Kind:      n.Kind,
		Message:   n.Message,
		ReadAt:    nullTime(n.ReadAt),
		CreatedAt: n.CreatedAt,
	}
}

// ListNotifications lista as notificações do usuário autenticado.
//
// @Summary   Lista as notificações do usuário
// @Tags      notifications
// @Produce   json
// @Security  Bearer
// @Param     unread  query     bool  false  "Apenas não-lidas"
// @Param     limit   query     int   false  "Tamanho da página (máx. 50)"
// @Param     offset  query     int   false  "Deslocamento"
// @Success   200     {object}  notificationListResponse
// @Failure   401     {object}  errorEnvelope
// @Router    /notifications [get]
func (h *Handler) ListNotifications(c *gin.Context) {
	limit, offset := parsePagination(c)
	unread, _ := strconv.ParseBool(c.Query("unread"))

	notifications, err := h.notifications.List(c.Request.Context(), actor(c), unread, limit, offset)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]notificationResponse, 0, len(notifications))
	for _, n := range notifications {
		out = append(out, toNotificationResponse(n))
	}
	c.JSON(http.StatusOK, gin.H{"notifications": out})
}

// UnreadNotificationCount devolve o número de notificações não-lidas (badge).
//
// @Summary   Contador de notificações não-lidas
// @Tags      notifications
// @Produce   json
// @Security  Bearer
// @Success   200  {object}  unreadCountResponse
// @Failure   401  {object}  errorEnvelope
// @Router    /notifications/unread_count [get]
func (h *Handler) UnreadNotificationCount(c *gin.Context) {
	count, err := h.notifications.UnreadCount(c.Request.Context(), actor(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkNotificationRead marca uma notificação como lida (idempotente).
//
// @Summary   Marca uma notificação como lida
// @Tags      notifications
// @Produce   json
// @Security  Bearer
// @Param     id   path  int  true  "ID da notificação"
// @Success   204  "marcada como lida"
// @Failure   400  {object}  errorEnvelope
// @Failure   404  {object}  errorEnvelope
// @Router    /notifications/{id}/read [post]
func (h *Handler) MarkNotificationRead(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.notifications.MarkRead(c.Request.Context(), actor(c), id); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// MarkAllNotificationsRead marca todas as notificações do usuário como lidas.
//
// @Summary   Marca todas as notificações como lidas
// @Tags      notifications
// @Produce   json
// @Security  Bearer
// @Success   204  "todas marcadas como lidas"
// @Failure   401  {object}  errorEnvelope
// @Router    /notifications/read_all [post]
func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	if err := h.notifications.MarkAllRead(c.Request.Context(), actor(c)); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
