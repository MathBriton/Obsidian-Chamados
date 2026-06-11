package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

func TestAddComment_GeneratesNotifications(t *testing.T) {
	e := newSLAEnv(t)
	ctx := context.Background()

	agent, err := e.store.Users.Create(ctx, repositories.CreateUserInput{
		TenantID: e.admin.TenantID, Name: "Ag", Email: "ag@acme.com", PasswordHash: "x", Role: services.RoleAgent,
	})
	if err != nil {
		t.Fatalf("seed agent: %v", err)
	}

	// Ticket aberto pelo customer, atribuído ao agente.
	tk, err := e.tickets.Create(ctx, e.customer, services.CreateTicketInput{
		Title: "X", Description: "y", Priority: "medium", CategoryID: e.category,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	aid := agent.ID
	if _, err := e.tickets.Update(ctx, e.admin, tk.ID, services.UpdateTicketInput{AssignedTo: &aid}); err != nil {
		t.Fatalf("assign: %v", err)
	}

	unread := func(userID int64) int64 {
		n, err := e.store.Notifications.CountUnread(ctx, e.admin.TenantID, userID)
		if err != nil {
			t.Fatalf("CountUnread: %v", err)
		}
		return n
	}

	// Admin (staff) comenta publicamente: notifica criador (customer) e
	// responsável (agente), mas não o autor (admin).
	if _, err := e.tickets.AddComment(ctx, e.admin, tk.ID, "olá", false); err != nil {
		t.Fatalf("AddComment público: %v", err)
	}
	if got := unread(e.customer.UserID); got != 1 {
		t.Fatalf("criador deveria ter 1 notificação, tem %d", got)
	}
	if got := unread(agent.ID); got != 1 {
		t.Fatalf("responsável deveria ter 1 notificação, tem %d", got)
	}
	if got := unread(e.admin.UserID); got != 0 {
		t.Fatalf("autor não deveria se autonotificar, tem %d", got)
	}

	// Nota interna não gera notificação.
	if _, err := e.tickets.AddComment(ctx, e.admin, tk.ID, "nota", true); err != nil {
		t.Fatalf("AddComment interna: %v", err)
	}
	if got := unread(e.customer.UserID); got != 1 {
		t.Fatalf("nota interna não deveria notificar; criador tem %d", got)
	}
}

func TestAddComment_AuthorIsCreatorNoSelfNotify(t *testing.T) {
	e := newSLAEnv(t)
	ctx := context.Background()

	// Customer abre e comenta no próprio ticket (sem responsável): ninguém é
	// notificado (autor == criador, sem assignee).
	tk, err := e.tickets.Create(ctx, e.customer, services.CreateTicketInput{
		Title: "X", Description: "y", Priority: "medium", CategoryID: e.category,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := e.tickets.AddComment(ctx, e.customer, tk.ID, "comentei", false); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	n, err := e.store.Notifications.CountUnread(ctx, e.customer.TenantID, e.customer.UserID)
	if err != nil {
		t.Fatalf("CountUnread: %v", err)
	}
	if n != 0 {
		t.Fatalf("autor-criador não deveria se autonotificar, tem %d", n)
	}
}

func TestNotifications_MarkReadIsolationAndIdempotency(t *testing.T) {
	e := newSLAEnv(t)
	ctx := context.Background()
	notifSvc := services.NewNotificationService(e.store)

	// Gera uma notificação para o customer (comentário público do admin).
	tk, err := e.tickets.Create(ctx, e.customer, services.CreateTicketInput{
		Title: "X", Description: "y", Priority: "medium", CategoryID: e.category,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := e.tickets.AddComment(ctx, e.admin, tk.ID, "oi", false); err != nil {
		t.Fatalf("AddComment: %v", err)
	}
	list, err := notifSvc.List(ctx, e.customer, true, 50, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("esperado 1 notificação não-lida; err=%v len=%d", err, len(list))
	}
	id := list[0].ID

	// Admin (outro usuário) não pode marcar a notificação do customer: 404.
	if err := notifSvc.MarkRead(ctx, e.admin, id); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("esperado ErrNotFound ao marcar notificação alheia, obtido %v", err)
	}

	// Dono marca como lida; re-marcar é idempotente.
	if err := notifSvc.MarkRead(ctx, e.customer, id); err != nil {
		t.Fatalf("MarkRead dono: %v", err)
	}
	if err := notifSvc.MarkRead(ctx, e.customer, id); err != nil {
		t.Fatalf("MarkRead idempotente: %v", err)
	}
	if c, _ := notifSvc.UnreadCount(ctx, e.customer); c != 0 {
		t.Fatalf("após marcar lida, não-lidas deveriam ser 0, são %d", c)
	}
}
