package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// Papéis usados nas regras de autorização (RBAC) de tickets.
const (
	RoleAdmin    = "admin"
	RoleAgent    = "agent"
	RoleCustomer = "customer"
)

// Status de resolução/fechamento que carimbam timestamps.
const (
	statusResolved = "resolved"
	statusClosed   = "closed"
)

// defaultListLimit limita o tamanho de página quando o cliente não informa.
const defaultListLimit = 50

// Actor é a identidade autenticada que executa a operação, extraída do access
// token (RNF01). A camada de service confia nesses dados, nunca no cliente.
type Actor struct {
	UserID   int64
	TenantID int64
	Role     string
}

func (a Actor) isStaff() bool { return a.Role == RoleAdmin || a.Role == RoleAgent }

// TicketService implementa as regras de negócio de tickets, incluindo o RBAC:
// customer só enxerga os próprios; agent/admin enxergam todos do tenant.
type TicketService struct {
	store *repositories.Store
}

// NewTicketService monta o serviço de tickets.
func NewTicketService(store *repositories.Store) *TicketService {
	return &TicketService{store: store}
}

// CreateTicketInput agrega os dados de abertura de um ticket. Priority é
// opcional (default "medium"); o status inicial é sempre "open".
type CreateTicketInput struct {
	Title       string
	Description string
	Priority    string
	CategoryID  int64
}

// Create abre um ticket no tenant do actor, em nome dele (created_by). Qualquer
// papel autenticado pode abrir. A categoria precisa existir no tenant.
func (s *TicketService) Create(ctx context.Context, actor Actor, in CreateTicketInput) (db.Ticket, error) {
	if _, err := s.store.Categories.GetByID(ctx, actor.TenantID, in.CategoryID); err != nil {
		return db.Ticket{}, invalidCategory(err)
	}

	priority := in.Priority
	if priority == "" {
		priority = "medium"
	}

	return s.store.Tickets.Create(ctx, repositories.CreateTicketInput{
		TenantID:    actor.TenantID,
		Title:       in.Title,
		Description: in.Description,
		Status:      "open",
		Priority:    priority,
		CategoryID:  in.CategoryID,
		CreatedBy:   actor.UserID,
	})
}

// Get devolve um ticket aplicando a visibilidade do actor. Um customer que
// tenta acessar ticket alheio recebe ErrNotFound, não revelando sua existência
// (ADR-003).
func (s *TicketService) Get(ctx context.Context, actor Actor, id int64) (db.Ticket, error) {
	ticket, err := s.store.Tickets.GetByID(ctx, actor.TenantID, id)
	if err != nil {
		return db.Ticket{}, err
	}
	if !actor.isStaff() && ticket.CreatedBy != actor.UserID {
		return db.Ticket{}, models.ErrNotFound
	}
	return ticket, nil
}

// List devolve os tickets visíveis ao actor: todos do tenant para staff,
// apenas os criados por ele para customer.
func (s *TicketService) List(ctx context.Context, actor Actor, limit, offset int64) ([]db.Ticket, error) {
	if limit <= 0 || limit > defaultListLimit {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	if actor.isStaff() {
		return s.store.Tickets.ListByTenant(ctx, actor.TenantID, limit, offset)
	}
	return s.store.Tickets.ListByCreator(ctx, actor.TenantID, actor.UserID, limit, offset)
}

// UpdateTicketInput descreve uma atualização parcial: campos nil ficam
// inalterados. Distinguir "não informado" de "valor zero" exige ponteiros.
type UpdateTicketInput struct {
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	CategoryID  *int64
	AssignedTo  *int64
}

// Update aplica uma atualização parcial respeitando o RBAC:
//   - customer: só o próprio ticket e apenas título/descrição;
//   - agent/admin: qualquer ticket do tenant, incluindo status, prioridade,
//     categoria e atribuição.
//
// Transições para "resolved"/"closed" carimbam resolved_at/closed_at.
func (s *TicketService) Update(ctx context.Context, actor Actor, id int64, in UpdateTicketInput) (db.Ticket, error) {
	ticket, err := s.Get(ctx, actor, id) // já aplica visibilidade (404 p/ alheio)
	if err != nil {
		return db.Ticket{}, err
	}

	if !actor.isStaff() {
		// Customer não pode mexer em status, prioridade, categoria ou atribuição.
		if in.Status != nil || in.Priority != nil || in.CategoryID != nil || in.AssignedTo != nil {
			return db.Ticket{}, models.ErrForbidden
		}
	}

	if in.Title != nil {
		ticket.Title = *in.Title
	}
	if in.Description != nil {
		ticket.Description = *in.Description
	}
	if in.Priority != nil {
		ticket.Priority = *in.Priority
	}
	if in.CategoryID != nil {
		if _, err := s.store.Categories.GetByID(ctx, actor.TenantID, *in.CategoryID); err != nil {
			return db.Ticket{}, invalidCategory(err)
		}
		ticket.CategoryID = *in.CategoryID
	}
	if in.AssignedTo != nil {
		if _, err := s.store.Users.GetByID(ctx, actor.TenantID, *in.AssignedTo); err != nil {
			return db.Ticket{}, invalidAssignee(err)
		}
		ticket.AssignedTo = sql.NullInt64{Int64: *in.AssignedTo, Valid: true}
	}
	if in.Status != nil {
		applyStatus(&ticket, *in.Status)
	}

	return s.store.Tickets.Update(ctx, ticket)
}

// AddComment registra um comentário num ticket. Exige que o actor enxergue o
// ticket (customer só nos próprios, senão 404). Notas internas (is_internal)
// só podem ser criadas por staff; um customer que tente isso recebe 403.
func (s *TicketService) AddComment(ctx context.Context, actor Actor, ticketID int64, body string, internal bool) (db.Comment, error) {
	if _, err := s.Get(ctx, actor, ticketID); err != nil {
		return db.Comment{}, err
	}
	if internal && !actor.isStaff() {
		return db.Comment{}, models.ErrForbidden
	}
	return s.store.Comments.Create(ctx, repositories.CreateCommentInput{
		TenantID:   actor.TenantID,
		TicketID:   ticketID,
		AuthorID:   actor.UserID,
		Body:       body,
		IsInternal: internal,
	})
}

// ListComments devolve os comentários visíveis de um ticket. Exige visibilidade
// do ticket; customers veem apenas comentários públicos, staff vê também as
// notas internas.
func (s *TicketService) ListComments(ctx context.Context, actor Actor, ticketID int64) ([]db.Comment, error) {
	if _, err := s.Get(ctx, actor, ticketID); err != nil {
		return nil, err
	}
	return s.store.Comments.ListByTicket(ctx, actor.TenantID, ticketID, actor.isStaff())
}

// applyStatus muda o status e carimba resolved_at/closed_at ao entrar nesses
// estados pela primeira vez.
func applyStatus(t *db.Ticket, status string) {
	t.Status = status
	now := time.Now().UTC()
	switch status {
	case statusResolved:
		if !t.ResolvedAt.Valid {
			t.ResolvedAt = sql.NullTime{Time: now, Valid: true}
		}
	case statusClosed:
		if !t.ClosedAt.Valid {
			t.ClosedAt = sql.NullTime{Time: now, Valid: true}
		}
	}
}

// invalidCategory converte um ErrNotFound da categoria em ErrInvalidCategory,
// preservando outros erros.
func invalidCategory(err error) error {
	if errors.Is(err, models.ErrNotFound) {
		return models.ErrInvalidCategory
	}
	return err
}

// invalidAssignee converte um ErrNotFound do responsável em ErrInvalidAssignee.
func invalidAssignee(err error) error {
	if errors.Is(err, models.ErrNotFound) {
		return models.ErrInvalidAssignee
	}
	return err
}
