package services

import (
	"context"
	"errors"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// TeamService gere as equipes de atendimento de um tenant e seus membros.
// As restrições de papel (listar: staff; criar/gerir membros: admin) são
// aplicadas na borda HTTP (middleware RequireRole).
type TeamService struct {
	store *repositories.Store
}

// NewTeamService monta o serviço de equipes.
func NewTeamService(store *repositories.Store) *TeamService {
	return &TeamService{store: store}
}

// TeamMember é a visão mínima de um integrante de equipe.
type TeamMember struct {
	UserID int64
	Name   string
	Role   string
}

// TeamWithMembers agrega uma equipe e seus integrantes ativos.
type TeamWithMembers struct {
	Team    db.Team
	Members []TeamMember
}

// Create cria uma equipe no tenant. Nome duplicado retorna ErrTeamTaken.
func (s *TeamService) Create(ctx context.Context, tenantID int64, name string) (db.Team, error) {
	return s.store.Teams.Create(ctx, tenantID, name)
}

// List devolve as equipes do tenant com seus membros ativos, em ordem
// alfabética.
func (s *TeamService) List(ctx context.Context, tenantID int64) ([]TeamWithMembers, error) {
	teams, err := s.store.Teams.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	links, err := s.store.Teams.ListMembersByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	byTeam := make(map[int64][]TeamMember, len(teams))
	for _, l := range links {
		byTeam[l.TeamID] = append(byTeam[l.TeamID], TeamMember{UserID: l.ID, Name: l.Name, Role: l.Role})
	}

	out := make([]TeamWithMembers, 0, len(teams))
	for _, t := range teams {
		members := byTeam[t.ID]
		if members == nil {
			members = []TeamMember{}
		}
		out = append(out, TeamWithMembers{Team: t, Members: members})
	}
	return out, nil
}

// AddMember vincula um usuário do tenant à equipe. Apenas staff ativo
// (admin/agent) pode integrar equipes; usuário inexistente, inativo ou
// customer retorna ErrInvalidMember. Equipe inexistente retorna ErrNotFound
// (ADR-003). A operação é idempotente.
func (s *TeamService) AddMember(ctx context.Context, tenantID, teamID, userID int64) error {
	if _, err := s.store.Teams.GetByID(ctx, tenantID, teamID); err != nil {
		return err
	}
	user, err := s.store.Users.GetByID(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrInvalidMember
		}
		return err
	}
	if !user.IsActive || (user.Role != RoleAdmin && user.Role != RoleAgent) {
		return models.ErrInvalidMember
	}
	return s.store.Teams.AddMember(ctx, tenantID, teamID, userID)
}

// RemoveMember desvincula um usuário da equipe. Equipe inexistente retorna
// ErrNotFound; remover quem não é membro é um no-op (idempotente).
func (s *TeamService) RemoveMember(ctx context.Context, tenantID, teamID, userID int64) error {
	if _, err := s.store.Teams.GetByID(ctx, tenantID, teamID); err != nil {
		return err
	}
	return s.store.Teams.RemoveMember(ctx, tenantID, teamID, userID)
}
