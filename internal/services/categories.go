package services

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// CategoryService gere as categorias de um tenant. A restrição de criação a
// admins é aplicada na borda HTTP (middleware RequireRole).
type CategoryService struct {
	store *repositories.Store
}

// NewCategoryService monta o serviço de categorias.
func NewCategoryService(store *repositories.Store) *CategoryService {
	return &CategoryService{store: store}
}

// Create cria uma categoria no tenant. Nome duplicado retorna ErrCategoryTaken.
func (s *CategoryService) Create(ctx context.Context, tenantID int64, name string) (db.Category, error) {
	return s.store.Categories.Create(ctx, tenantID, name)
}

// List devolve as categorias do tenant em ordem alfabética.
func (s *CategoryService) List(ctx context.Context, tenantID int64) ([]db.Category, error) {
	return s.store.Categories.ListByTenant(ctx, tenantID)
}
