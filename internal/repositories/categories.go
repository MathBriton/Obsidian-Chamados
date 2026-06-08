package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
)

// CategoryRepository dá acesso à tabela categories, escopada por tenant_id
// (ADR-001).
type CategoryRepository struct {
	q db.Querier
}

// Create insere uma categoria no tenant. Retorna models.ErrCategoryTaken se o
// par (tenant_id, name) já existir.
func (r *CategoryRepository) Create(ctx context.Context, tenantID int64, name string) (db.Category, error) {
	cat, err := r.q.CreateCategory(ctx, db.CreateCategoryParams{TenantID: tenantID, Name: name})
	if isUniqueViolation(err, "categories.name") {
		return db.Category{}, models.ErrCategoryTaken
	}
	return cat, err
}

// GetByID busca uma categoria pelo id dentro do tenant.
func (r *CategoryRepository) GetByID(ctx context.Context, tenantID, id int64) (db.Category, error) {
	cat, err := r.q.GetCategoryByID(ctx, db.GetCategoryByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		return db.Category{}, notFound(err)
	}
	return cat, nil
}

// ListByTenant lista as categorias do tenant em ordem alfabética.
func (r *CategoryRepository) ListByTenant(ctx context.Context, tenantID int64) ([]db.Category, error) {
	return r.q.ListCategoriesByTenant(ctx, tenantID)
}
