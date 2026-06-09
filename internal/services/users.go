package services

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// UserService implementa a gestão de usuários de um tenant pelo admin:
// criação (agents/customers/admins), listagem e desativação.
type UserService struct {
	store *repositories.Store
}

// NewUserService monta o serviço de usuários.
func NewUserService(store *repositories.Store) *UserService {
	return &UserService{store: store}
}

// CreateUserInput agrega os dados para criar um usuário no tenant.
type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     string
}

// Create adiciona um usuário ao tenant com a senha já hasheada (RNF03).
// Email duplicado no tenant retorna models.ErrEmailTaken.
func (s *UserService) Create(ctx context.Context, tenantID int64, in CreateUserInput) (db.User, error) {
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return db.User{}, err
	}
	return s.store.Users.Create(ctx, repositories.CreateUserInput{
		TenantID:     tenantID,
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: hash,
		Role:         in.Role,
	})
}

// List devolve os usuários do tenant, paginado.
func (s *UserService) List(ctx context.Context, tenantID, limit, offset int64) ([]db.User, error) {
	if limit <= 0 || limit > defaultListLimit {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.Users.ListByTenant(ctx, tenantID, limit, offset)
}

// Deactivate desativa um usuário do tenant. Um admin não pode desativar a si
// mesmo (evita lockout). Usuário inexistente no tenant retorna ErrNotFound.
func (s *UserService) Deactivate(ctx context.Context, tenantID, actorID, targetID int64) error {
	if actorID == targetID {
		return models.ErrForbidden
	}
	if _, err := s.store.Users.GetByID(ctx, tenantID, targetID); err != nil {
		return err
	}
	return s.store.Users.Deactivate(ctx, tenantID, targetID)
}
