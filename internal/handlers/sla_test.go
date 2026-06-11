package handlers_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// seedSLA insere uma política de SLA diretamente no store (tenants criados via
// seedTenant não passam pelo registro, que é quem semeia os defaults).
func (e env) seedSLA(t *testing.T, tenantID int64, priority string, firstResp, resolution int64) {
	t.Helper()
	if _, err := e.store.SLA.Upsert(context.Background(), repositories.UpsertPolicyInput{
		TenantID: tenantID, Priority: priority, FirstResponseMins: firstResp, ResolutionMins: resolution,
	}); err != nil {
		t.Fatalf("seedSLA: %v", err)
	}
}

func TestListSLAPolicies_StaffSeesPolicies(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	e.seedSLA(t, tn.ID, "high", 30, 240)
	agent := e.seedUser(t, tn.ID, "a@acme.com", services.RoleAgent)

	w := do(t, e.r, http.MethodGet, "/sla-policies", "", e.token(t, agent))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	policies := decode(t, w)["policies"].([]interface{})
	if len(policies) != 1 {
		t.Fatalf("esperado 1 política, obtido %d", len(policies))
	}
}

func TestListSLAPolicies_CustomerForbidden(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)

	w := do(t, e.r, http.MethodGet, "/sla-policies", "", e.token(t, customer))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, esperado 403", w.Code)
	}
}

func TestUpsertSLAPolicy_AdminUpserts(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)

	w := do(t, e.r, http.MethodPut, "/sla-policies/high",
		`{"first_response_mins":15,"resolution_mins":90}`, e.token(t, admin))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["priority"] != "high" || int64(body["resolution_mins"].(float64)) != 90 {
		t.Fatalf("resposta inesperada: %v", body)
	}

	// Segundo PUT atualiza (upsert) em vez de duplicar.
	w = do(t, e.r, http.MethodPut, "/sla-policies/high",
		`{"first_response_mins":10,"resolution_mins":60}`, e.token(t, admin))
	if w.Code != http.StatusOK {
		t.Fatalf("segundo PUT status %d, esperado 200", w.Code)
	}
	got, err := e.store.SLA.GetByPriority(context.Background(), tn.ID, "high")
	if err != nil {
		t.Fatalf("GetByPriority: %v", err)
	}
	if got.ResolutionMins != 60 {
		t.Fatalf("resolution_mins %d, esperado 60 (upsert)", got.ResolutionMins)
	}
}

func TestUpsertSLAPolicy_AgentForbidden(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	agent := e.seedUser(t, tn.ID, "a@acme.com", services.RoleAgent)

	w := do(t, e.r, http.MethodPut, "/sla-policies/high",
		`{"first_response_mins":15,"resolution_mins":90}`, e.token(t, agent))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, esperado 403", w.Code)
	}
}

func TestUpsertSLAPolicy_InvalidPriority(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)

	w := do(t, e.r, http.MethodPut, "/sla-policies/urgente",
		`{"first_response_mins":15,"resolution_mins":90}`, e.token(t, admin))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, esperado 400", w.Code)
	}
}

func TestUpsertSLAPolicy_InvalidBody(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)

	w := do(t, e.r, http.MethodPut, "/sla-policies/high",
		`{"first_response_mins":0,"resolution_mins":90}`, e.token(t, admin))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, esperado 400", w.Code)
	}
}

func TestCreateTicket_ResponseIncludesSLA(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	e.seedSLA(t, tn.ID, "medium", 60, 480)
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)

	id := e.createTicket(t, e.token(t, customer), cat.ID, "Falha")
	w := do(t, e.r, http.MethodGet, "/tickets/"+strconv.FormatInt(id, 10), "", e.token(t, customer))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200", w.Code)
	}
	body := decode(t, w)
	if body["resolution_due_at"] == nil {
		t.Fatal("resolution_due_at deveria estar preenchido")
	}
	if body["sla_resolution_state"] != services.SLAStateOk {
		t.Fatalf("sla_resolution_state %v, esperado ok", body["sla_resolution_state"])
	}
}

func TestTicket_ClosedWithoutResolveCountsAsMet(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	e.seedSLA(t, tn.ID, "medium", 60, 480)
	cat := e.seedCategory(t, tn.ID, "Bugs")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	tok := e.token(t, admin)

	id := e.createTicket(t, tok, cat.ID, "Duplicado")
	// Fecha direto, sem passar por "resolved" (resolved_at fica nulo).
	w := do(t, e.r, http.MethodPatch, "/tickets/"+strconv.FormatInt(id, 10),
		`{"status":"closed"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	// Fechado antes do prazo de resolução => SLA de resolução cumprido.
	if body["resolved_at"] != nil {
		t.Fatalf("resolved_at deveria ser nulo, obtido %v", body["resolved_at"])
	}
	if body["sla_resolution_state"] != services.SLAStateMet {
		t.Fatalf("sla_resolution_state %v, esperado met (fechado dentro do prazo)", body["sla_resolution_state"])
	}
}
