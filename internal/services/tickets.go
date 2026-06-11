package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// Tipos de evento do histórico (auditoria) de tickets.
const (
	eventCreated         = "created"
	eventStatusChanged   = "status_changed"
	eventPriorityChanged = "priority_changed"
	eventCategoryChanged = "category_changed"
	eventAssigneeChanged = "assignee_changed"
	eventTeamChanged     = "team_changed"
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
// papel autenticado pode abrir. A categoria precisa existir no tenant. A
// abertura entra no histórico como evento "created".
func (s *TicketService) Create(ctx context.Context, actor Actor, in CreateTicketInput) (db.Ticket, error) {
	if _, err := s.store.Categories.GetByID(ctx, actor.TenantID, in.CategoryID); err != nil {
		return db.Ticket{}, invalidCategory(err)
	}

	priority := in.Priority
	if priority == "" {
		priority = "medium"
	}

	// Prazos de SLA a partir da abertura. Sem política para a prioridade, os
	// prazos ficam nulos (estado "none").
	frDue, resDue, err := s.slaDueDatesFor(ctx, actor.TenantID, priority, time.Now().UTC())
	if err != nil {
		return db.Ticket{}, err
	}

	var ticket db.Ticket
	err = s.store.ExecTx(ctx, func(r *repositories.Repositories) error {
		var err error
		ticket, err = r.Tickets.Create(ctx, repositories.CreateTicketInput{
			TenantID:           actor.TenantID,
			Title:              in.Title,
			Description:        in.Description,
			Status:             "open",
			Priority:           priority,
			CategoryID:         in.CategoryID,
			CreatedBy:          actor.UserID,
			FirstResponseDueAt: frDue,
			ResolutionDueAt:    resDue,
		})
		if err != nil {
			return err
		}
		_, err = r.Events.Create(ctx, repositories.CreateEventInput{
			TenantID: actor.TenantID, TicketID: ticket.ID, ActorID: actor.UserID, Kind: eventCreated,
		})
		return err
	})
	return ticket, err
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
// apenas os criados por ele para customer. O filtro é aplicado dentro desse
// escopo de visibilidade (um customer nunca amplia o que enxerga filtrando).
func (s *TicketService) List(ctx context.Context, actor Actor, f repositories.TicketFilter, limit, offset int64) ([]db.Ticket, error) {
	if limit <= 0 || limit > defaultListLimit {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	if actor.isStaff() {
		return s.store.Tickets.ListByTenant(ctx, actor.TenantID, f, limit, offset)
	}
	return s.store.Tickets.ListByCreator(ctx, actor.TenantID, actor.UserID, f, limit, offset)
}

// Stats devolve as contagens de tickets visíveis ao actor, no mesmo escopo de
// visibilidade da listagem: todo o tenant para staff, apenas os próprios para
// customer.
func (s *TicketService) Stats(ctx context.Context, actor Actor) (repositories.TicketStats, error) {
	if actor.isStaff() {
		return s.store.Tickets.Stats(ctx, actor.TenantID, nil)
	}
	return s.store.Tickets.Stats(ctx, actor.TenantID, &actor.UserID)
}

// UpdateTicketInput descreve uma atualização parcial: campos nil ficam
// inalterados. Distinguir "não informado" de "valor zero" exige ponteiros.
type UpdateTicketInput struct {
	Title          *string
	Description    *string
	Status         *string
	Priority       *string
	CategoryID     *int64
	AssignedTo     *int64
	AssignedTeamID *int64
}

// Update aplica uma atualização parcial respeitando o RBAC:
//   - customer: só o próprio ticket e apenas título/descrição;
//   - agent/admin: qualquer ticket do tenant, incluindo status, prioridade,
//     categoria e atribuição.
//
// Transições para "resolved"/"closed" carimbam resolved_at/closed_at. Mudanças
// de status, prioridade, categoria, responsável e equipe entram no histórico
// (ticket_events) na mesma transação do UPDATE, com snapshots legíveis.
func (s *TicketService) Update(ctx context.Context, actor Actor, id int64, in UpdateTicketInput) (db.Ticket, error) {
	ticket, err := s.Get(ctx, actor, id) // já aplica visibilidade (404 p/ alheio)
	if err != nil {
		return db.Ticket{}, err
	}

	if !actor.isStaff() {
		// Customer não pode mexer em status, prioridade, categoria ou atribuição.
		if in.Status != nil || in.Priority != nil || in.CategoryID != nil || in.AssignedTo != nil || in.AssignedTeamID != nil {
			return db.Ticket{}, models.ErrForbidden
		}
	}

	var events []repositories.CreateEventInput
	record := func(kind string, oldValue, newValue *string) {
		events = append(events, repositories.CreateEventInput{
			TenantID: actor.TenantID, TicketID: ticket.ID, ActorID: actor.UserID,
			Kind: kind, OldValue: oldValue, NewValue: newValue,
		})
	}

	if in.Title != nil {
		ticket.Title = *in.Title
	}
	if in.Description != nil {
		ticket.Description = *in.Description
	}
	if in.Priority != nil && *in.Priority != ticket.Priority {
		record(eventPriorityChanged, ptr(ticket.Priority), in.Priority)
		ticket.Priority = *in.Priority
		// Recalcula os prazos de SLA sobre a abertura com a nova política
		// (sem política => prazos nulos). A primeira resposta já dada é
		// preservada.
		frDue, resDue, err := s.slaDueDatesFor(ctx, actor.TenantID, ticket.Priority, ticket.CreatedAt)
		if err != nil {
			return db.Ticket{}, err
		}
		ticket.FirstResponseDueAt = frDue
		ticket.ResolutionDueAt = resDue
	}
	if in.CategoryID != nil && *in.CategoryID != ticket.CategoryID {
		newCat, err := s.store.Categories.GetByID(ctx, actor.TenantID, *in.CategoryID)
		if err != nil {
			return db.Ticket{}, invalidCategory(err)
		}
		record(eventCategoryChanged, ptr(s.categoryName(ctx, actor.TenantID, ticket.CategoryID)), ptr(newCat.Name))
		ticket.CategoryID = *in.CategoryID
	}
	if in.AssignedTo != nil && (!ticket.AssignedTo.Valid || ticket.AssignedTo.Int64 != *in.AssignedTo) {
		newUser, err := s.store.Users.GetByID(ctx, actor.TenantID, *in.AssignedTo)
		if err != nil {
			return db.Ticket{}, invalidAssignee(err)
		}
		var oldName *string
		if ticket.AssignedTo.Valid {
			oldName = ptr(s.userName(ctx, actor.TenantID, ticket.AssignedTo.Int64))
		}
		record(eventAssigneeChanged, oldName, ptr(newUser.Name))
		ticket.AssignedTo = sql.NullInt64{Int64: *in.AssignedTo, Valid: true}
	}
	if in.AssignedTeamID != nil && (!ticket.AssignedTeamID.Valid || ticket.AssignedTeamID.Int64 != *in.AssignedTeamID) {
		newTeam, err := s.store.Teams.GetByID(ctx, actor.TenantID, *in.AssignedTeamID)
		if err != nil {
			return db.Ticket{}, invalidTeam(err)
		}
		var oldName *string
		if ticket.AssignedTeamID.Valid {
			oldName = ptr(s.teamName(ctx, actor.TenantID, ticket.AssignedTeamID.Int64))
		}
		record(eventTeamChanged, oldName, ptr(newTeam.Name))
		ticket.AssignedTeamID = sql.NullInt64{Int64: *in.AssignedTeamID, Valid: true}
	}
	if in.Status != nil && *in.Status != ticket.Status {
		record(eventStatusChanged, ptr(ticket.Status), in.Status)
		applyStatus(&ticket, *in.Status)
	}

	var updated db.Ticket
	err = s.store.ExecTx(ctx, func(r *repositories.Repositories) error {
		var err error
		updated, err = r.Tickets.Update(ctx, ticket)
		if err != nil {
			return err
		}
		for _, ev := range events {
			if _, err := r.Events.Create(ctx, ev); err != nil {
				return err
			}
		}
		return nil
	})
	return updated, err
}

// ListEvents devolve o histórico de um ticket em ordem cronológica, com a
// mesma visibilidade do Get (customer só nos próprios; alheio responde 404).
func (s *TicketService) ListEvents(ctx context.Context, actor Actor, ticketID int64) ([]repositories.EventWithActor, error) {
	if _, err := s.Get(ctx, actor, ticketID); err != nil {
		return nil, err
	}
	return s.store.Events.ListByTicket(ctx, actor.TenantID, ticketID)
}

// categoryName devolve um snapshot legível para o histórico; se a busca
// falhar, degrada para "#id" em vez de abortar a operação.
func (s *TicketService) categoryName(ctx context.Context, tenantID, id int64) string {
	if cat, err := s.store.Categories.GetByID(ctx, tenantID, id); err == nil {
		return cat.Name
	}
	return fmt.Sprintf("#%d", id)
}

func (s *TicketService) userName(ctx context.Context, tenantID, id int64) string {
	if u, err := s.store.Users.GetByID(ctx, tenantID, id); err == nil {
		return u.Name
	}
	return fmt.Sprintf("#%d", id)
}

func (s *TicketService) teamName(ctx context.Context, tenantID, id int64) string {
	if t, err := s.store.Teams.GetByID(ctx, tenantID, id); err == nil {
		return t.Name
	}
	return fmt.Sprintf("#%d", id)
}

// slaDueDatesFor busca a política de SLA da prioridade e calcula os prazos a
// partir da base (abertura do ticket). Sem política, devolve prazos nulos
// (ticket sem SLA) — não é erro.
func (s *TicketService) slaDueDatesFor(ctx context.Context, tenantID int64, priority string, base time.Time) (firstResponse, resolution sql.NullTime, err error) {
	policy, err := s.store.SLA.GetByPriority(ctx, tenantID, priority)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return sql.NullTime{}, sql.NullTime{}, nil
		}
		return sql.NullTime{}, sql.NullTime{}, err
	}
	firstResponse, resolution = slaDueDates(policy, base)
	return firstResponse, resolution, nil
}

// ptr devolve um ponteiro para o valor (atalho para montar eventos).
func ptr(s string) *string { return &s }

// AddComment registra um comentário num ticket. Exige que o actor enxergue o
// ticket (customer só nos próprios, senão 404). Notas internas (is_internal)
// só podem ser criadas por staff; um customer que tente isso recebe 403.
func (s *TicketService) AddComment(ctx context.Context, actor Actor, ticketID int64, body string, internal bool) (repositories.CommentWithAuthor, error) {
	ticket, err := s.Get(ctx, actor, ticketID)
	if err != nil {
		return repositories.CommentWithAuthor{}, err
	}
	if internal && !actor.isStaff() {
		return repositories.CommentWithAuthor{}, models.ErrForbidden
	}

	// Primeira resposta do SLA: primeiro comentário público de staff. Notas
	// internas não contam e a marcação é idempotente (só carimba se ainda nulo).
	stampFirstResponse := actor.isStaff() && !internal && !ticket.FirstRespondedAt.Valid

	var comment db.Comment
	err = s.store.ExecTx(ctx, func(r *repositories.Repositories) error {
		var err error
		comment, err = r.Comments.Create(ctx, repositories.CreateCommentInput{
			TenantID:   actor.TenantID,
			TicketID:   ticketID,
			AuthorID:   actor.UserID,
			Body:       body,
			IsInternal: internal,
		})
		if err != nil {
			return err
		}
		if stampFirstResponse {
			return r.Tickets.StampFirstResponse(ctx, actor.TenantID, ticketID, time.Now().UTC())
		}
		return nil
	})
	if err != nil {
		return repositories.CommentWithAuthor{}, err
	}
	return repositories.CommentWithAuthor{Comment: comment, AuthorName: s.userName(ctx, actor.TenantID, actor.UserID)}, nil
}

// ListComments devolve os comentários visíveis de um ticket. Exige visibilidade
// do ticket; customers veem apenas comentários públicos, staff vê também as
// notas internas.
func (s *TicketService) ListComments(ctx context.Context, actor Actor, ticketID int64) ([]repositories.CommentWithAuthor, error) {
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

// invalidTeam converte um ErrNotFound da equipe em ErrInvalidTeam.
func invalidTeam(err error) error {
	if errors.Is(err, models.ErrNotFound) {
		return models.ErrInvalidTeam
	}
	return err
}
