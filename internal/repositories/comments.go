package repositories

import (
	"context"
	"time"

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

// CommentWithAuthor anexa o nome do autor (JOIN em users) ao comentário.
type CommentWithAuthor struct {
	db.Comment
	AuthorName string
}

// ListByTicket lista os comentários de um ticket em ordem cronológica, com o
// nome do autor. Quando includeInternal é false, retorna apenas os
// comentários públicos — notas internas ficam restritas a staff.
func (r *CommentRepository) ListByTicket(ctx context.Context, tenantID, ticketID int64, includeInternal bool) ([]CommentWithAuthor, error) {
	if includeInternal {
		rows, err := r.q.ListCommentsByTicket(ctx, db.ListCommentsByTicketParams{TenantID: tenantID, TicketID: ticketID})
		if err != nil {
			return nil, err
		}
		out := make([]CommentWithAuthor, 0, len(rows))
		for _, row := range rows {
			out = append(out, CommentWithAuthor{Comment: rowComment(row.ID, row.TenantID, row.TicketID, row.AuthorID, row.Body, row.IsInternal, row.CreatedAt), AuthorName: row.AuthorName})
		}
		return out, nil
	}
	rows, err := r.q.ListPublicCommentsByTicket(ctx, db.ListPublicCommentsByTicketParams{TenantID: tenantID, TicketID: ticketID})
	if err != nil {
		return nil, err
	}
	out := make([]CommentWithAuthor, 0, len(rows))
	for _, row := range rows {
		out = append(out, CommentWithAuthor{Comment: rowComment(row.ID, row.TenantID, row.TicketID, row.AuthorID, row.Body, row.IsInternal, row.CreatedAt), AuthorName: row.AuthorName})
	}
	return out, nil
}

// rowComment remonta um db.Comment a partir das colunas das rows geradas pelo
// SQLC (os dois SELECTs com JOIN produzem tipos distintos de mesma forma).
func rowComment(id, tenantID, ticketID, authorID int64, body string, isInternal bool, createdAt time.Time) db.Comment {
	return db.Comment{
		ID: id, TenantID: tenantID, TicketID: ticketID, AuthorID: authorID,
		Body: body, IsInternal: isInternal, CreatedAt: createdAt,
	}
}
