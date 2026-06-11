package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
)

// TeamRepository dá acesso às tabelas teams e team_members, escopadas por
// tenant_id (ADR-001).
type TeamRepository struct {
	q db.Querier
}

// Create insere uma equipe no tenant. Retorna models.ErrTeamTaken se o par
// (tenant_id, name) já existir.
func (r *TeamRepository) Create(ctx context.Context, tenantID int64, name string) (db.Team, error) {
	team, err := r.q.CreateTeam(ctx, db.CreateTeamParams{TenantID: tenantID, Name: name})
	if isUniqueViolation(err, "teams.name") {
		return db.Team{}, models.ErrTeamTaken
	}
	return team, err
}

// GetByID busca uma equipe pelo id dentro do tenant.
func (r *TeamRepository) GetByID(ctx context.Context, tenantID, id int64) (db.Team, error) {
	team, err := r.q.GetTeamByID(ctx, db.GetTeamByIDParams{TenantID: tenantID, ID: id})
	if err != nil {
		return db.Team{}, notFound(err)
	}
	return team, nil
}

// ListByTenant lista as equipes do tenant em ordem alfabética.
func (r *TeamRepository) ListByTenant(ctx context.Context, tenantID int64) ([]db.Team, error) {
	return r.q.ListTeamsByTenant(ctx, tenantID)
}

// AddMember vincula um usuário à equipe (idempotente: re-adicionar não falha).
func (r *TeamRepository) AddMember(ctx context.Context, tenantID, teamID, userID int64) error {
	return r.q.AddTeamMember(ctx, db.AddTeamMemberParams{TeamID: teamID, UserID: userID, TenantID: tenantID})
}

// RemoveMember desvincula um usuário da equipe (idempotente).
func (r *TeamRepository) RemoveMember(ctx context.Context, tenantID, teamID, userID int64) error {
	return r.q.RemoveTeamMember(ctx, db.RemoveTeamMemberParams{TenantID: tenantID, TeamID: teamID, UserID: userID})
}

// ListMembersByTenant lista os vínculos equipe-usuário de todo o tenant
// (apenas usuários ativos), para o service agrupar por equipe.
func (r *TeamRepository) ListMembersByTenant(ctx context.Context, tenantID int64) ([]db.ListTeamMembersByTenantRow, error) {
	return r.q.ListTeamMembersByTenant(ctx, tenantID)
}
