package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
)

// TenantRepository dá acesso à tabela tenants — a raiz do isolamento.
type TenantRepository struct {
	q db.Querier
}

// Create insere um novo tenant. Retorna models.ErrSlugTaken se o slug já
// estiver em uso.
func (r *TenantRepository) Create(ctx context.Context, name, slug string) (db.Tenant, error) {
	tenant, err := r.q.CreateTenant(ctx, db.CreateTenantParams{Name: name, Slug: slug})
	if isUniqueViolation(err, "tenants.slug") {
		return db.Tenant{}, models.ErrSlugTaken
	}
	return tenant, err
}

// GetBySlug busca um tenant pelo slug. Retorna models.ErrNotFound se ausente.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (db.Tenant, error) {
	tenant, err := r.q.GetTenantBySlug(ctx, slug)
	if err != nil {
		return db.Tenant{}, notFound(err)
	}
	return tenant, nil
}

// GetByID busca um tenant pelo id. Retorna models.ErrNotFound se ausente.
func (r *TenantRepository) GetByID(ctx context.Context, id int64) (db.Tenant, error) {
	tenant, err := r.q.GetTenantByID(ctx, id)
	if err != nil {
		return db.Tenant{}, notFound(err)
	}
	return tenant, nil
}
