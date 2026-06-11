package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// listEvents devolve o histórico de um ticket via API.
func (e env) listEvents(t *testing.T, bearer string, ticketID int64) []map[string]any {
	t.Helper()
	w := do(t, e.r, http.MethodGet, fmt.Sprintf("/tickets/%d/events", ticketID), "", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /tickets/%d/events: status %d; body=%s", ticketID, w.Code, w.Body.String())
	}
	raw := decode(t, w)["events"].([]any)
	events := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		events = append(events, item.(map[string]any))
	}
	return events
}

func TestTicketEvents_RecordsLifecycle(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	agentTok := e.token(t, agent)

	id := e.createTicket(t, e.token(t, customer), cat.ID, "Sem rede")

	// Atribui responsável + prioridade, depois resolve.
	e.patchTicket(t, agentTok, id, fmt.Sprintf(`{"assigned_to":%d,"priority":"high"}`, agent.ID))
	e.patchTicket(t, agentTok, id, `{"status":"resolved"}`)

	events := e.listEvents(t, agentTok, id)
	kinds := make([]string, 0, len(events))
	for _, ev := range events {
		kinds = append(kinds, ev["kind"].(string))
	}

	want := []string{"created", "priority_changed", "assignee_changed", "status_changed"}
	if len(kinds) != len(want) {
		t.Fatalf("eventos %v, esperado %v", kinds, want)
	}
	for _, w := range want {
		found := false
		for _, k := range kinds {
			if k == w {
				found = true
			}
		}
		if !found {
			t.Fatalf("evento %q ausente em %v", w, kinds)
		}
	}

	// O evento de status guarda o snapshot old/new.
	for _, ev := range events {
		if ev["kind"] == "status_changed" {
			if ev["old_value"] != "open" || ev["new_value"] != "resolved" {
				t.Fatalf("status_changed com valores errados: %v", ev)
			}
		}
		if ev["kind"] == "assignee_changed" {
			if ev["old_value"] != nil || ev["new_value"] != "agent@acme.com" {
				t.Fatalf("assignee_changed com valores errados: %v", ev)
			}
		}
	}
}

func TestTicketEvents_NoChangeNoEvent(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	agentTok := e.token(t, agent)

	id := e.createTicket(t, agentTok, cat.ID, "Idempotente")

	// PATCH com o mesmo status/prioridade atuais não gera eventos novos.
	e.patchTicket(t, agentTok, id, `{"status":"open","priority":"medium"}`)

	events := e.listEvents(t, agentTok, id)
	if len(events) != 1 || events[0]["kind"] != "created" {
		t.Fatalf("esperava apenas o evento created, veio %v", events)
	}
}

func TestTicketEvents_VisibilityFollowsTicket(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)

	id := e.createTicket(t, e.token(t, custA), cat.ID, "privado")

	// Dono vê o histórico; outro customer recebe 404 (ADR-003).
	if got := e.listEvents(t, e.token(t, custA), id); len(got) != 1 {
		t.Fatalf("dono viu %d eventos, esperado 1", len(got))
	}
	w := do(t, e.r, http.MethodGet, fmt.Sprintf("/tickets/%d/events", id), "", e.token(t, custB))
	if w.Code != http.StatusNotFound {
		t.Fatalf("intruso: status %d, esperado 404", w.Code)
	}
}
