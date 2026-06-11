package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// Estados derivados de um prazo de SLA, calculados em tempo de leitura a partir
// dos timestamps do ticket e do instante atual — nunca armazenados.
const (
	SLAStateNone     = "none"     // a prioridade não tem política de SLA
	SLAStateMet      = "met"      // cumprido dentro do prazo
	SLAStateOk       = "ok"       // pendente, dentro do prazo e sem risco
	SLAStateAtRisk   = "at_risk"  // pendente, restando ≤20% da janela
	SLAStateBreached = "breached" // venceu sem cumprir (ou cumprido após o prazo)
)

// slaAtRiskFraction define a fração final da janela em que o prazo é
// considerado "em risco" (faltando 20% ou menos do tempo total).
const slaAtRiskFraction = 0.2

// DefaultSLAPolicies são as políticas semeadas para todo tenant novo (no
// registro). Mantêm paridade com o backfill da migration 0003_sla. Minutos de
// primeira resposta / resolução, da prioridade mais alta para a mais baixa.
var DefaultSLAPolicies = []struct {
	Priority          string
	FirstResponseMins int64
	ResolutionMins    int64
}{
	{"critical", 15, 120},
	{"high", 30, 240},
	{"medium", 60, 480},
	{"low", 120, 1440},
}

// SLAService gere as políticas de SLA por prioridade de um tenant. As
// restrições de papel (ler: staff; editar: admin) são aplicadas na borda HTTP.
type SLAService struct {
	store *repositories.Store
}

// NewSLAService monta o serviço de SLA.
func NewSLAService(store *repositories.Store) *SLAService {
	return &SLAService{store: store}
}

// List devolve as políticas de SLA do tenant, da prioridade mais alta para a
// mais baixa.
func (s *SLAService) List(ctx context.Context, tenantID int64) ([]db.SlaPolicy, error) {
	return s.store.SLA.ListByTenant(ctx, tenantID)
}

// Upsert cria ou atualiza a política de uma prioridade. A validação de
// prioridade e dos minutos (> 0) é feita na borda HTTP.
func (s *SLAService) Upsert(ctx context.Context, tenantID int64, priority string, firstResponseMins, resolutionMins int64) (db.SlaPolicy, error) {
	return s.store.SLA.Upsert(ctx, repositories.UpsertPolicyInput{
		TenantID:          tenantID,
		Priority:          priority,
		FirstResponseMins: firstResponseMins,
		ResolutionMins:    resolutionMins,
	})
}

// slaDueDates calcula os prazos de primeira resposta e de resolução a partir de
// uma base (criação do ticket) somada aos minutos da política.
func slaDueDates(policy db.SlaPolicy, base time.Time) (firstResponse, resolution sql.NullTime) {
	firstResponse = sql.NullTime{Time: base.Add(time.Duration(policy.FirstResponseMins) * time.Minute), Valid: true}
	resolution = sql.NullTime{Time: base.Add(time.Duration(policy.ResolutionMins) * time.Minute), Valid: true}
	return firstResponse, resolution
}

// SLAState deriva o estado de um prazo. dueAt é o vencimento; doneAt é o
// instante em que a meta foi cumprida (primeira resposta ou resolução); base é
// a abertura do ticket (início da janela); now é o instante de avaliação.
func SLAState(dueAt, doneAt sql.NullTime, base, now time.Time) string {
	if !dueAt.Valid {
		return SLAStateNone
	}
	if doneAt.Valid {
		if doneAt.Time.After(dueAt.Time) {
			return SLAStateBreached
		}
		return SLAStateMet
	}
	if now.After(dueAt.Time) {
		return SLAStateBreached
	}
	total := dueAt.Time.Sub(base)
	remaining := dueAt.Time.Sub(now)
	if total > 0 && remaining <= time.Duration(float64(total)*slaAtRiskFraction) {
		return SLAStateAtRisk
	}
	return SLAStateOk
}
