package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

func TestComments_CustomerSeesOnlyPublic(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	id := e.createTicket(t, e.token(t, customer), cat.ID, "ticket")
	path := fmt.Sprintf("/tickets/%d/comments", id)

	// Agent adiciona um comentário público e uma nota interna.
	if w := do(t, e.r, http.MethodPost, path, `{"body":"ola, estamos vendo"}`, e.token(t, agent)); w.Code != http.StatusCreated {
		t.Fatalf("agent comenta publico: status %d", w.Code)
	}
	if w := do(t, e.r, http.MethodPost, path, `{"body":"nota interna","is_internal":true}`, e.token(t, agent)); w.Code != http.StatusCreated {
		t.Fatalf("agent nota interna: status %d", w.Code)
	}

	// Customer vê apenas o comentário público.
	wc := do(t, e.r, http.MethodGet, path, "", e.token(t, customer))
	if got := len(decode(t, wc)["comments"].([]any)); got != 1 {
		t.Fatalf("customer viu %d comentarios, esperado 1 (so publico)", got)
	}
	// Agent vê os dois.
	wa := do(t, e.r, http.MethodGet, path, "", e.token(t, agent))
	if got := len(decode(t, wa)["comments"].([]any)); got != 2 {
		t.Fatalf("agent viu %d comentarios, esperado 2", got)
	}
}

func TestComments_CustomerCannotCreateInternal(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	tok := e.token(t, customer)
	id := e.createTicket(t, tok, cat.ID, "ticket")
	path := fmt.Sprintf("/tickets/%d/comments", id)

	// Nota interna é proibida para customer.
	if w := do(t, e.r, http.MethodPost, path, `{"body":"x","is_internal":true}`, tok); w.Code != http.StatusForbidden {
		t.Fatalf("customer nota interna: status %d, esperado 403", w.Code)
	}
	// Comentário público no próprio ticket é permitido.
	if w := do(t, e.r, http.MethodPost, path, `{"body":"obrigado"}`, tok); w.Code != http.StatusCreated {
		t.Fatalf("customer comenta proprio: status %d, esperado 201", w.Code)
	}
}

func TestComments_RequireTicketVisibility(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)
	id := e.createTicket(t, e.token(t, custA), cat.ID, "secreto")
	path := fmt.Sprintf("/tickets/%d/comments", id)

	// Customer B não enxerga o ticket → comentar e listar dão 404.
	if w := do(t, e.r, http.MethodPost, path, `{"body":"intruso"}`, e.token(t, custB)); w.Code != http.StatusNotFound {
		t.Fatalf("custB comenta alheio: status %d, esperado 404", w.Code)
	}
	if w := do(t, e.r, http.MethodGet, path, "", e.token(t, custB)); w.Code != http.StatusNotFound {
		t.Fatalf("custB lista alheio: status %d, esperado 404", w.Code)
	}
}
