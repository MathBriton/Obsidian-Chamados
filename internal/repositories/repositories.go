// Package repositories envolve o código gerado pelo SQLC, reforçando o
// isolamento multi-tenant (ADR-001) e traduzindo erros do banco para os
// erros de domínio em internal/models.
package repositories

import "github.com/MathBriton/Obsidian-Chamados/internal/db"

// Repositories agrupa os repositórios que compartilham a mesma fonte de
// dados — o pool de conexões ou uma transação. Construir os repositórios a
// partir de um db.Querier permite reaproveitá-los dentro de Store.ExecTx.
type Repositories struct {
	Users      *UserRepository
	Tenants    *TenantRepository
	Tokens     *RefreshTokenRepository
	Categories *CategoryRepository
	Tickets    *TicketRepository
	Comments   *CommentRepository
	Teams      *TeamRepository
	Events     *TicketEventRepository
	SLA        *SLARepository
}

// New monta os repositórios sobre um db.Querier (pool ou transação).
func New(q db.Querier) *Repositories {
	return &Repositories{
		Users:      &UserRepository{q: q},
		Tenants:    &TenantRepository{q: q},
		Tokens:     &RefreshTokenRepository{q: q},
		Categories: &CategoryRepository{q: q},
		Tickets:    &TicketRepository{q: q},
		Comments:   &CommentRepository{q: q},
		Teams:      &TeamRepository{q: q},
		Events:     &TicketEventRepository{q: q},
		SLA:        &SLARepository{q: q},
	}
}
