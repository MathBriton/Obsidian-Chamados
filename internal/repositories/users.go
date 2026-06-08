package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
)

// UserRepository dá acesso à tabela users. Toda operação é escopada por
// tenant_id, reforçando o isolamento na camada de dados (ADR-001).
type UserRepository struct {
	q db.Querier
}

// CreateUserInput agrega os dados para criar um usuário.
type CreateUserInput struct {
	TenantID     int64
	Name         string
	Email        string
	PasswordHash string
	Role         string
}

// Create insere um usuário no tenant. Retorna models.ErrEmailTaken se o par
// (tenant_id, email) já existir.
func (r *UserRepository) Create(ctx context.Context, in CreateUserInput) (db.User, error) {
	user, err := r.q.CreateUser(ctx, db.CreateUserParams{
		TenantID:     in.TenantID,
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: in.PasswordHash,
		Role:         in.Role,
	})
	if isUniqueViolation(err, "users.email") {
		return db.User{}, models.ErrEmailTaken
	}
	return user, err
}

// GetByEmail busca um usuário pelo email dentro do tenant.
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID int64, email string) (db.User, error) {
	user, err := r.q.GetUserByEmail(ctx, db.GetUserByEmailParams{TenantID: tenantID, Email: email})
	if err != nil {
		return db.User{}, notFound(err)
	}
	return user, nil
}

// GetByID busca um usuário pelo id dentro do tenant.
func (r *UserRepository) GetByID(ctx context.Context, tenantID, id int64) (db.User, error) {
	user, err := r.q.GetUserByID(ctx, db.GetUserByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		return db.User{}, notFound(err)
	}
	return user, nil
}

// ListByTenant lista usuários do tenant, paginado.
func (r *UserRepository) ListByTenant(ctx context.Context, tenantID, limit, offset int64) ([]db.User, error) {
	return r.q.ListUsersByTenant(ctx, db.ListUsersByTenantParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
}

// Deactivate marca um usuário como inativo dentro do tenant.
func (r *UserRepository) Deactivate(ctx context.Context, tenantID, id int64) error {
	return r.q.DeactivateUser(ctx, db.DeactivateUserParams{TenantID: tenantID, ID: id})
}
