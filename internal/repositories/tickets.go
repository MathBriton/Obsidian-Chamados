package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// TicketRepository dá acesso à tabela tickets, escopada por tenant_id (ADR-001).
type TicketRepository struct {
	q db.Querier
}

// CreateTicketInput agrega os dados para abrir um ticket. status e priority já
// chegam validados/normalizados pela camada de service.
type CreateTicketInput struct {
	TenantID    int64
	Title       string
	Description string
	Status      string
	Priority    string
	CategoryID  int64
	CreatedBy   int64
}

// Create abre um ticket no tenant.
func (r *TicketRepository) Create(ctx context.Context, in CreateTicketInput) (db.Ticket, error) {
	return r.q.CreateTicket(ctx, db.CreateTicketParams{
		TenantID:    in.TenantID,
		Title:       in.Title,
		Description: in.Description,
		Status:      in.Status,
		Priority:    in.Priority,
		CategoryID:  in.CategoryID,
		CreatedBy:   in.CreatedBy,
	})
}

// GetByID busca um ticket pelo id dentro do tenant.
func (r *TicketRepository) GetByID(ctx context.Context, tenantID, id int64) (db.Ticket, error) {
	ticket, err := r.q.GetTicketByID(ctx, db.GetTicketByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		return db.Ticket{}, notFound(err)
	}
	return ticket, nil
}

// TicketFilter restringe a listagem de tickets. Campos nil não filtram. Search
// casa por substring (case-insensitive) em título ou descrição.
type TicketFilter struct {
	Status     *string
	Priority   *string
	AssignedTo *int64
	Search     *string
}

// ListByTenant lista todos os tickets do tenant, filtrado e paginado (visão de
// agent/admin).
func (r *TicketRepository) ListByTenant(ctx context.Context, tenantID int64, f TicketFilter, limit, offset int64) ([]db.Ticket, error) {
	return r.q.ListTicketsByTenant(ctx, db.ListTicketsByTenantParams{
		TenantID:   tenantID,
		Status:     nullable(f.Status),
		Priority:   nullable(f.Priority),
		AssignedTo: nullable(f.AssignedTo),
		Search:     nullable(f.Search),
		Limit:      limit,
		Offset:     offset,
	})
}

// ListByCreator lista os tickets criados por um usuário no tenant, filtrado e
// paginado (visão de customer).
func (r *TicketRepository) ListByCreator(ctx context.Context, tenantID, createdBy int64, f TicketFilter, limit, offset int64) ([]db.Ticket, error) {
	return r.q.ListTicketsByCreator(ctx, db.ListTicketsByCreatorParams{
		TenantID:   tenantID,
		CreatedBy:  createdBy,
		Status:     nullable(f.Status),
		Priority:   nullable(f.Priority),
		AssignedTo: nullable(f.AssignedTo),
		Search:     nullable(f.Search),
		Limit:      limit,
		Offset:     offset,
	})
}

// nullable converte um ponteiro opcional no parâmetro esperado pelo SQLC para
// sqlc.narg: nil quando ausente, o valor quando presente.
func nullable[T any](p *T) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// TicketStats agrega as contagens de tickets de um escopo (tenant inteiro ou
// apenas os criados por um usuário). Mapas trazem só as chaves com ocorrência;
// o zero-fill dos enums fica na camada de apresentação.
type TicketStats struct {
	ByStatus         map[string]int64
	ByPriority       map[string]int64
	UnassignedActive int64
}

// Stats calcula as contagens de tickets do tenant. Com createdBy não-nil, o
// escopo é restrito aos tickets criados por esse usuário (visão de customer).
func (r *TicketRepository) Stats(ctx context.Context, tenantID int64, createdBy *int64) (TicketStats, error) {
	stats := TicketStats{ByStatus: map[string]int64{}, ByPriority: map[string]int64{}}

	byStatus, err := r.q.CountTicketsByStatus(ctx, db.CountTicketsByStatusParams{
		TenantID: tenantID, CreatedBy: nullable(createdBy),
	})
	if err != nil {
		return TicketStats{}, err
	}
	for _, row := range byStatus {
		stats.ByStatus[row.Status] = row.Total
	}

	byPriority, err := r.q.CountTicketsByPriority(ctx, db.CountTicketsByPriorityParams{
		TenantID: tenantID, CreatedBy: nullable(createdBy),
	})
	if err != nil {
		return TicketStats{}, err
	}
	for _, row := range byPriority {
		stats.ByPriority[row.Priority] = row.Total
	}

	stats.UnassignedActive, err = r.q.CountUnassignedActiveTickets(ctx, db.CountUnassignedActiveTicketsParams{
		TenantID: tenantID, CreatedBy: nullable(createdBy),
	})
	if err != nil {
		return TicketStats{}, err
	}
	return stats, nil
}

// Update persiste o ticket inteiro (read-modify-write feito no service). O
// filtro por tenant_id garante o isolamento mesmo no UPDATE.
func (r *TicketRepository) Update(ctx context.Context, t db.Ticket) (db.Ticket, error) {
	updated, err := r.q.UpdateTicket(ctx, db.UpdateTicketParams{
		Title:          t.Title,
		Description:    t.Description,
		Status:         t.Status,
		Priority:       t.Priority,
		CategoryID:     t.CategoryID,
		AssignedTo:     t.AssignedTo,
		AssignedTeamID: t.AssignedTeamID,
		ResolvedAt:     t.ResolvedAt,
		ClosedAt:       t.ClosedAt,
		TenantID:       t.TenantID,
		ID:             t.ID,
	})
	if err != nil {
		return db.Ticket{}, notFound(err)
	}
	return updated, nil
}
