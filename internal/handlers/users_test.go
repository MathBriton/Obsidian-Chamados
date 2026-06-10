package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

func TestCreateUser_AdminCreatesAgent(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)

	body := `{"name":"Ana Agente","email":"ana@acme.com","password":"senha-forte","role":"agent"}`
	w := do(t, e.r, http.MethodPost, "/users", body, e.token(t, admin))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado 201; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if got["role"] != "agent" || got["is_active"] != true {
		t.Fatalf("campos inesperados: %v", got)
	}
	// Não pode vazar o hash de senha.
	if _, ok := got["password_hash"]; ok {
		t.Fatal("resposta vazou password_hash")
	}
}

func TestCreateUser_NonAdminForbidden(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)

	body := `{"name":"X","email":"x@acme.com","password":"senha-forte","role":"customer"}`
	w := do(t, e.r, http.MethodPost, "/users", body, e.token(t, agent))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, esperado 403", w.Code)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	tok := e.token(t, admin)

	body := `{"name":"Dup","email":"dup@acme.com","password":"senha-forte","role":"customer"}`
	if w := do(t, e.r, http.MethodPost, "/users", body, tok); w.Code != http.StatusCreated {
		t.Fatalf("primeira criacao: status %d", w.Code)
	}
	w := do(t, e.r, http.MethodPost, "/users", body, tok)
	if w.Code != http.StatusConflict {
		t.Fatalf("status %d, esperado 409", w.Code)
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "email_taken" {
		t.Fatalf("code %v, esperado email_taken", code)
	}
}

func TestListUsers_AdminSeesTenantUsers(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)

	w := do(t, e.r, http.MethodGet, "/users", "", e.token(t, admin))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200", w.Code)
	}
	if got := len(decode(t, w)["users"].([]any)); got != 2 {
		t.Fatalf("listou %d usuarios, esperado 2", got)
	}
}

func TestDeactivateUser_Success(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	target := e.seedUser(t, tn.ID, "alvo@acme.com", services.RoleCustomer)
	tok := e.token(t, admin)

	if w := do(t, e.r, http.MethodDelete, fmt.Sprintf("/users/%d", target.ID), "", tok); w.Code != http.StatusNoContent {
		t.Fatalf("deactivate: status %d, esperado 204", w.Code)
	}

	// O usuário aparece como inativo na listagem.
	w := do(t, e.r, http.MethodGet, "/users", "", tok)
	users := decode(t, w)["users"].([]any)
	for _, u := range users {
		m := u.(map[string]any)
		if int64(m["id"].(float64)) == target.ID && m["is_active"] != false {
			t.Fatalf("usuario %d deveria estar inativo", target.ID)
		}
	}
}

func TestDeactivateUser_CannotDeactivateSelf(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)

	w := do(t, e.r, http.MethodDelete, fmt.Sprintf("/users/%d", admin.ID), "", e.token(t, admin))
	if w.Code != http.StatusForbidden {
		t.Fatalf("auto-desativacao: status %d, esperado 403", w.Code)
	}
}

func TestListAssignees_StaffOnlyActiveStaff(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	e.seedUser(t, tn.ID, "cliente@acme.com", services.RoleCustomer) // não atribuível
	inactive := e.seedUser(t, tn.ID, "ex@acme.com", services.RoleAgent)
	if err := e.store.Users.Deactivate(context.Background(), tn.ID, inactive.ID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	// Agent pode listar os atribuíveis.
	w := do(t, e.r, http.MethodGet, "/assignees", "", e.token(t, agent))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200", w.Code)
	}
	users := decode(t, w)["users"].([]any)
	if len(users) != 2 {
		t.Fatalf("listou %d atribuiveis, esperado 2 (admin + agent ativos)", len(users))
	}
	// Confere que customer e inativo ficaram de fora.
	ids := map[int64]bool{}
	for _, u := range users {
		ids[int64(u.(map[string]any)["id"].(float64))] = true
	}
	if !ids[admin.ID] || !ids[agent.ID] {
		t.Fatalf("esperava admin (%d) e agent (%d) na lista", admin.ID, agent.ID)
	}
}

func TestListAssignees_CustomerForbidden(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)

	w := do(t, e.r, http.MethodGet, "/assignees", "", e.token(t, customer))
	if w.Code != http.StatusForbidden {
		t.Fatalf("status %d, esperado 403", w.Code)
	}
}

func TestDeactivateUser_NotFound(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)

	w := do(t, e.r, http.MethodDelete, "/users/9999", "", e.token(t, admin))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d, esperado 404", w.Code)
	}
}
