package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// CommentRepository dá acesso à tabela comments, escopada por tenant_id
// (ADR-001).
type CommentRepository struct {
	q db.Querier
}

// CreateCommentInput agrega os dados para registrar um comentário.
type CreateCommentInput struct {
	TenantID   int64
	TicketID   int64
	AuthorID   int64
	Body       string
	IsInternal bool
}

// Create registra um comentário no ticket.
func (r *CommentRepository) Create(ctx context.Context, in CreateCommentInput) (db.Comment, error) {
	return r.q.CreateComment(ctx, db.CreateCommentParams{
		TenantID:   in.TenantID,
		TicketID:   in.TicketID,
		AuthorID:   in.AuthorID,
		Body:       in.Body,
		IsInternal: in.IsInternal,
	})
}

// ListByTicket lista os comentários de um ticket em ordem cronológica.
// Quando includeInternal é false, retorna apenas os comentários públicos —
// notas internas ficam restritas a staff.
func (r *CommentRepository) ListByTicket(ctx context.Context, tenantID, ticketID int64, includeInternal bool) ([]db.Comment, error) {
	if includeInternal {
		return r.q.ListCommentsByTicket(ctx, db.ListCommentsByTicketParams{TenantID: tenantID, TicketID: ticketID})
	}
	return r.q.ListPublicCommentsByTicket(ctx, db.ListPublicCommentsByTicketParams{TenantID: tenantID, TicketID: ticketID})
}
