package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// getStats chama GET /stats e devolve o corpo decodificado.
func (e env) getStats(t *testing.T, bearer string) map[string]any {
	t.Helper()
	w := do(t, e.r, http.MethodGet, "/stats", "", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /stats: status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	return decode(t, w)
}

func TestGetStats_StaffSeesTenantWide(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	agentTok := e.token(t, agent)

	idLogin := e.createTicket(t, e.token(t, custA), cat.ID, "Falha no login")
	idEmail := e.createTicket(t, e.token(t, custA), cat.ID, "Email fora do ar")
	e.createTicket(t, e.token(t, custB), cat.ID, "Impressora")

	// Um resolvido com prioridade alta; um atribuído ao agent (segue ativo).
	e.patchTicket(t, agentTok, idLogin, `{"status":"resolved","priority":"high"}`)
	e.patchTicket(t, agentTok, idEmail, fmt.Sprintf(`{"assigned_to":%d}`, agent.ID))

	stats := e.getStats(t, agentTok)

	if got := stats["total"].(float64); got != 3 {
		t.Fatalf("total %v, esperado 3", got)
	}
	byStatus := stats["by_status"].(map[string]any)
	if byStatus["open"].(float64) != 2 || byStatus["resolved"].(float64) != 1 {
		t.Fatalf("by_status errado: %v", byStatus)
	}
	// Enum zero-filled: status sem tickets aparecem com 0.
	if byStatus["closed"].(float64) != 0 {
		t.Fatalf("by_status deveria zero-preencher 'closed': %v", byStatus)
	}
	byPriority := stats["by_priority"].(map[string]any)
	if byPriority["medium"].(float64) != 2 || byPriority["high"].(float64) != 1 {
		t.Fatalf("by_priority errado: %v", byPriority)
	}
	// Ativos sem responsável: só "Impressora" (login foi resolvido, email atribuído).
	if got := stats["unassigned_active"].(float64); got != 1 {
		t.Fatalf("unassigned_active %v, esperado 1", got)
	}
}

func TestGetStats_CustomerSeesOnlyOwn(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)

	e.createTicket(t, e.token(t, custA), cat.ID, "do A 1")
	e.createTicket(t, e.token(t, custA), cat.ID, "do A 2")
	e.createTicket(t, e.token(t, custB), cat.ID, "do B")

	stats := e.getStats(t, e.token(t, custA))
	if got := stats["total"].(float64); got != 2 {
		t.Fatalf("customer A: total %v, esperado 2 (só os próprios)", got)
	}
}

func TestGetStats_CrossTenantIsolation(t *testing.T) {
	e := newEnv(t)
	tn1 := e.seedTenant(t, "T1", "t1")
	cat := e.seedCategory(t, tn1.ID, "Bugs")
	owner := e.seedUser(t, tn1.ID, "o@t1.com", services.RoleCustomer)
	e.createTicket(t, e.token(t, owner), cat.ID, "do-tenant-1")

	tn2 := e.seedTenant(t, "T2", "t2")
	admin2 := e.seedUser(t, tn2.ID, "x@t2.com", services.RoleAdmin)

	stats := e.getStats(t, e.token(t, admin2))
	if got := stats["total"].(float64); got != 0 {
		t.Fatalf("admin do T2 viu %v tickets do T1, esperado 0 (ADR-001)", got)
	}
}
