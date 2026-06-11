package repositories

import (
	"context"
	"database/sql"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// TicketEventRepository dá acesso à tabela ticket_events (histórico de
// auditoria), escopada por tenant_id (ADR-001).
type TicketEventRepository struct {
	q db.Querier
}

// CreateEventInput agrega os dados de um evento de ticket. OldValue/NewValue
// nil indicam ausência de valor (ex.: ticket ainda sem responsável).
type CreateEventInput struct {
	TenantID int64
	TicketID int64
	ActorID  int64
	Kind     string
	OldValue *string
	NewValue *string
}

// Create registra um evento no histórico do ticket.
func (r *TicketEventRepository) Create(ctx context.Context, in CreateEventInput) (db.TicketEvent, error) {
	return r.q.CreateTicketEvent(ctx, db.CreateTicketEventParams{
		TenantID: in.TenantID,
		TicketID: in.TicketID,
		ActorID:  in.ActorID,
		Kind:     in.Kind,
		OldValue: nullString(in.OldValue),
		NewValue: nullString(in.NewValue),
	})
}

// EventWithActor anexa o nome do autor da ação (JOIN em users) ao evento.
type EventWithActor struct {
	db.TicketEvent
	ActorName string
}

// ListByTicket devolve o histórico de um ticket em ordem cronológica, com o
// nome de quem executou cada ação.
func (r *TicketEventRepository) ListByTicket(ctx context.Context, tenantID, ticketID int64) ([]EventWithActor, error) {
	rows, err := r.q.ListTicketEvents(ctx, db.ListTicketEventsParams{TenantID: tenantID, TicketID: ticketID})
	if err != nil {
		return nil, err
	}
	out := make([]EventWithActor, 0, len(rows))
	for _, row := range rows {
		out = append(out, EventWithActor{
			TicketEvent: db.TicketEvent{
				ID: row.ID, TenantID: row.TenantID, TicketID: row.TicketID, ActorID: row.ActorID,
				Kind: row.Kind, OldValue: row.OldValue, NewValue: row.NewValue, CreatedAt: row.CreatedAt,
			},
			ActorName: row.ActorName,
		})
	}
	return out, nil
}

func nullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}
