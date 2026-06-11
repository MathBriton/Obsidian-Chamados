package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// createTeam cria uma equipe via API e devolve o id.
func (e env) createTeam(t *testing.T, bearer, name string) int64 {
	t.Helper()
	w := do(t, e.r, http.MethodPost, "/teams", fmt.Sprintf(`{"name":%q}`, name), bearer)
	if w.Code != http.StatusCreated {
		t.Fatalf("createTeam %q: status %d, esperado 201; body=%s", name, w.Code, w.Body.String())
	}
	return int64(decode(t, w)["id"].(float64))
}

func TestCreateTeam_AdminOnlyAndUniqueName(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)

	e.createTeam(t, e.token(t, admin), "Infra")

	// Nome duplicado no tenant responde 409 team_taken.
	w := do(t, e.r, http.MethodPost, "/teams", `{"name":"Infra"}`, e.token(t, admin))
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicada: status %d, esperado 409; body=%s", w.Code, w.Body.String())
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "team_taken" {
		t.Fatalf("code %v, esperado team_taken", code)
	}

	// Agent não cria equipe.
	if w := do(t, e.r, http.MethodPost, "/teams", `{"name":"Suporte"}`, e.token(t, agent)); w.Code != http.StatusForbidden {
		t.Fatalf("agent cria equipe: status %d, esperado 403", w.Code)
	}
}

func TestTeamMembers_AddListRemove(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	adminTok := e.token(t, admin)

	teamID := e.createTeam(t, adminTok, "Infra")

	// Vincula o agent (204) e confere na listagem.
	w := do(t, e.r, http.MethodPost, fmt.Sprintf("/teams/%d/members", teamID),
		fmt.Sprintf(`{"user_id":%d}`, agent.ID), adminTok)
	if w.Code != http.StatusNoContent {
		t.Fatalf("add member: status %d, esperado 204; body=%s", w.Code, w.Body.String())
	}

	wList := do(t, e.r, http.MethodGet, "/teams", "", adminTok)
	if wList.Code != http.StatusOK {
		t.Fatalf("list teams: status %d", wList.Code)
	}
	teams := decode(t, wList)["teams"].([]any)
	if len(teams) != 1 {
		t.Fatalf("esperava 1 equipe, veio %d", len(teams))
	}
	members := teams[0].(map[string]any)["members"].([]any)
	if len(members) != 1 || int64(members[0].(map[string]any)["id"].(float64)) != agent.ID {
		t.Fatalf("membros errados: %v", members)
	}

	// Remove (204) e a listagem volta a ficar vazia.
	wDel := do(t, e.r, http.MethodDelete, fmt.Sprintf("/teams/%d/members/%d", teamID, agent.ID), "", adminTok)
	if wDel.Code != http.StatusNoContent {
		t.Fatalf("remove member: status %d, esperado 204", wDel.Code)
	}
	wList2 := do(t, e.r, http.MethodGet, "/teams", "", adminTok)
	members2 := decode(t, wList2)["teams"].([]any)[0].(map[string]any)["members"].([]any)
	if len(members2) != 0 {
		t.Fatalf("membro nao foi removido: %v", members2)
	}
}

func TestAddTeamMember_RejectsCustomerAndUnknown(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	adminTok := e.token(t, admin)
	teamID := e.createTeam(t, adminTok, "Infra")

	// Customer não pode integrar equipes (422 invalid_member).
	w := do(t, e.r, http.MethodPost, fmt.Sprintf("/teams/%d/members", teamID),
		fmt.Sprintf(`{"user_id":%d}`, customer.ID), adminTok)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("customer membro: status %d, esperado 422; body=%s", w.Code, w.Body.String())
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "invalid_member" {
		t.Fatalf("code %v, esperado invalid_member", code)
	}

	// Usuário inexistente também é invalid_member.
	w2 := do(t, e.r, http.MethodPost, fmt.Sprintf("/teams/%d/members", teamID), `{"user_id":999}`, adminTok)
	if w2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("membro inexistente: status %d, esperado 422", w2.Code)
	}

	// Equipe inexistente responde 404 (ADR-003).
	w3 := do(t, e.r, http.MethodPost, "/teams/999/members", `{"user_id":1}`, adminTok)
	if w3.Code != http.StatusNotFound {
		t.Fatalf("equipe inexistente: status %d, esperado 404", w3.Code)
	}
}

func TestListTeams_StaffOnlyAndTenantScoped(t *testing.T) {
	e := newEnv(t)
	tn1 := e.seedTenant(t, "T1", "t1")
	admin1 := e.seedUser(t, tn1.ID, "a@t1.com", services.RoleAdmin)
	customer1 := e.seedUser(t, tn1.ID, "c@t1.com", services.RoleCustomer)
	e.createTeam(t, e.token(t, admin1), "Infra-T1")

	tn2 := e.seedTenant(t, "T2", "t2")
	admin2 := e.seedUser(t, tn2.ID, "a@t2.com", services.RoleAdmin)

	// Customer não lista equipes.
	if w := do(t, e.r, http.MethodGet, "/teams", "", e.token(t, customer1)); w.Code != http.StatusForbidden {
		t.Fatalf("customer lista equipes: status %d, esperado 403", w.Code)
	}

	// Admin do T2 não enxerga equipes do T1 (ADR-001).
	w := do(t, e.r, http.MethodGet, "/teams", "", e.token(t, admin2))
	if got := len(decode(t, w)["teams"].([]any)); got != 0 {
		t.Fatalf("admin T2 viu %d equipes do T1, esperado 0", got)
	}
}

func TestUpdateTicket_AssignTeamAndFilter(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	agentTok := e.token(t, agent)

	teamID := e.createTeam(t, e.token(t, e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)), "Infra")
	idA := e.createTicket(t, e.token(t, customer), cat.ID, "com equipe")
	e.createTicket(t, e.token(t, customer), cat.ID, "sem equipe")

	// Staff atribui o ticket à equipe.
	w := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", idA),
		fmt.Sprintf(`{"assigned_team_id":%d}`, teamID), agentTok)
	if w.Code != http.StatusOK {
		t.Fatalf("atribuir equipe: status %d; body=%s", w.Code, w.Body.String())
	}
	if got := int64(decode(t, w)["assigned_team_id"].(float64)); got != teamID {
		t.Fatalf("assigned_team_id %v, esperado %d", got, teamID)
	}

	// Filtro por equipe devolve só o atribuído.
	titles := e.listTitles(t, agentTok, fmt.Sprintf("?team_id=%d", teamID))
	if len(titles) != 1 || titles[0] != "com equipe" {
		t.Fatalf("filtro team_id: %v", titles)
	}

	// Equipe inexistente responde 422 invalid_team.
	w2 := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", idA), `{"assigned_team_id":999}`, agentTok)
	if w2.Code != http.StatusUnprocessableEntity {
		t.Fatalf("equipe invalida: status %d, esperado 422", w2.Code)
	}
	if code := decode(t, w2)["error"].(map[string]any)["code"]; code != "invalid_team" {
		t.Fatalf("code %v, esperado invalid_team", code)
	}

	// Customer não pode atribuir equipe (403).
	w3 := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", idA),
		fmt.Sprintf(`{"assigned_team_id":%d}`, teamID), e.token(t, customer))
	if w3.Code != http.StatusForbidden {
		t.Fatalf("customer atribui equipe: status %d, esperado 403", w3.Code)
	}
}
