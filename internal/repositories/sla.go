package repositories

import (
	"context"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// SLARepository dá acesso à tabela sla_policies, escopada por tenant_id
// (ADR-001).
type SLARepository struct {
	q db.Querier
}

// UpsertPolicyInput agrega os dados de uma política de SLA por prioridade.
type UpsertPolicyInput struct {
	TenantID          int64
	Priority          string
	FirstResponseMins int64
	ResolutionMins    int64
}

// Upsert cria ou atualiza a política da prioridade no tenant (UNIQUE
// tenant_id+priority). Usado tanto pelo endpoint de admin quanto pelo seed de
// defaults no registro de um novo tenant.
func (r *SLARepository) Upsert(ctx context.Context, in UpsertPolicyInput) (db.SlaPolicy, error) {
	return r.q.UpsertSLAPolicy(ctx, db.UpsertSLAPolicyParams{
		TenantID:          in.TenantID,
		Priority:          in.Priority,
		FirstResponseMins: in.FirstResponseMins,
		ResolutionMins:    in.ResolutionMins,
	})
}

// GetByPriority busca a política de uma prioridade no tenant. Ausência retorna
// models.ErrNotFound (o service trata como "sem SLA").
func (r *SLARepository) GetByPriority(ctx context.Context, tenantID int64, priority string) (db.SlaPolicy, error) {
	policy, err := r.q.GetSLAPolicyByPriority(ctx, db.GetSLAPolicyByPriorityParams{TenantID: tenantID, Priority: priority})
	if err != nil {
		return db.SlaPolicy{}, notFound(err)
	}
	return policy, nil
}

// ListByTenant lista as políticas do tenant, da prioridade mais alta para a
// mais baixa.
func (r *SLARepository) ListByTenant(ctx context.Context, tenantID int64) ([]db.SlaPolicy, error) {
	return r.q.ListSLAPoliciesByTenant(ctx, tenantID)
}
