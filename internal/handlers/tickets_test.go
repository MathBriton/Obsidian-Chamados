package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/database/testdb"
	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/handlers"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// env é um ambiente de teste com acesso ao router, ao store (para semear
// dados) e ao token manager (para emitir access tokens de papéis arbitrários,
// já que o endpoint de registro só cria admins).
type env struct {
	r      *gin.Engine
	store  *repositories.Store
	tokens *auth.TokenManager
}

func newEnv(t *testing.T) env {
	t.Helper()
	store := repositories.NewStore(testdb.New(t))
	tm := auth.NewTokenManager("test-secret", 15*time.Minute)
	h := handlers.New(
		services.NewAuthService(store, tm, 7*24*time.Hour),
		services.NewCategoryService(store),
		services.NewTicketService(store),
		services.NewUserService(store),
		tm,
	)
	return env{r: h.Router(), store: store, tokens: tm}
}

func (e env) seedTenant(t *testing.T, name, slug string) db.Tenant {
	t.Helper()
	tenant, err := e.store.Tenants.Create(context.Background(), name, slug)
	if err != nil {
		t.Fatalf("seedTenant: %v", err)
	}
	return tenant
}

func (e env) seedUser(t *testing.T, tenantID int64, email, role string) db.User {
	t.Helper()
	user, err := e.store.Users.Create(context.Background(), repositories.CreateUserInput{
		TenantID: tenantID, Name: email, Email: email, PasswordHash: "x", Role: role,
	})
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return user
}

func (e env) seedCategory(t *testing.T, tenantID int64, name string) db.Category {
	t.Helper()
	cat, err := e.store.Categories.Create(context.Background(), tenantID, name)
	if err != nil {
		t.Fatalf("seedCategory: %v", err)
	}
	return cat
}

func (e env) token(t *testing.T, u db.User) string {
	t.Helper()
	tok, err := e.tokens.GenerateAccessToken(u.ID, u.TenantID, u.Role)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tok
}

// createTicket abre um ticket via API e devolve o id criado.
func (e env) createTicket(t *testing.T, bearer string, categoryID int64, title string) int64 {
	t.Helper()
	body := fmt.Sprintf(`{"title":%q,"description":"desc","category_id":%d}`, title, categoryID)
	w := do(t, e.r, http.MethodPost, "/tickets", body, bearer)
	if w.Code != http.StatusCreated {
		t.Fatalf("createTicket %q: status %d, esperado 201; body=%s", title, w.Code, w.Body.String())
	}
	return int64(decode(t, w)["id"].(float64))
}

func TestCreateTicket_CustomerOpensOwn(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)

	w := do(t, e.r, http.MethodPost, "/tickets",
		fmt.Sprintf(`{"title":"Falha no login","description":"nao consigo entrar","category_id":%d}`, cat.ID),
		e.token(t, customer))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado 201; body=%s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["status"] != "open" || body["priority"] != "medium" {
		t.Fatalf("defaults errados: %v", body)
	}
	if int64(body["created_by"].(float64)) != customer.ID {
		t.Fatalf("created_by %v, esperado %d", body["created_by"], customer.ID)
	}
}

func TestCreateTicket_InvalidCategory(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)

	w := do(t, e.r, http.MethodPost, "/tickets",
		`{"title":"X","description":"y","category_id":999}`, e.token(t, customer))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, esperado 422", w.Code)
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "invalid_category" {
		t.Fatalf("code %v, esperado invalid_category", code)
	}
}

func TestListTickets_RBAC(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)

	e.createTicket(t, e.token(t, custA), cat.ID, "ticket-A")
	e.createTicket(t, e.token(t, custB), cat.ID, "ticket-B")

	// Customer A enxerga apenas o próprio.
	wA := do(t, e.r, http.MethodGet, "/tickets", "", e.token(t, custA))
	if got := len(decode(t, wA)["tickets"].([]any)); got != 1 {
		t.Fatalf("customer A viu %d tickets, esperado 1", got)
	}

	// Agent enxerga todos do tenant.
	wAgent := do(t, e.r, http.MethodGet, "/tickets", "", e.token(t, agent))
	if got := len(decode(t, wAgent)["tickets"].([]any)); got != 2 {
		t.Fatalf("agent viu %d tickets, esperado 2", got)
	}
}

// patchTicket aplica um PATCH e exige sucesso (helper para montar cenários).
func (e env) patchTicket(t *testing.T, bearer string, id int64, body string) {
	t.Helper()
	if w := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", id), body, bearer); w.Code != http.StatusOK {
		t.Fatalf("patchTicket %d: status %d; body=%s", id, w.Code, w.Body.String())
	}
}

// listTitles devolve os títulos retornados por GET /tickets com a query dada.
func (e env) listTitles(t *testing.T, bearer, query string) []string {
	t.Helper()
	w := do(t, e.r, http.MethodGet, "/tickets"+query, "", bearer)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /tickets%s: status %d; body=%s", query, w.Code, w.Body.String())
	}
	raw := decode(t, w)["tickets"].([]any)
	titles := make([]string, 0, len(raw))
	for _, item := range raw {
		titles = append(titles, item.(map[string]any)["title"].(string))
	}
	return titles
}

func TestListTickets_Filters(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	agentTok := e.token(t, agent)

	idLogin := e.createTicket(t, e.token(t, customer), cat.ID, "Falha no login")
	idEmail := e.createTicket(t, e.token(t, customer), cat.ID, "Email fora do ar")
	e.createTicket(t, e.token(t, customer), cat.ID, "Impressora")

	e.patchTicket(t, agentTok, idLogin, `{"status":"resolved","priority":"high"}`)
	e.patchTicket(t, agentTok, idEmail, fmt.Sprintf(`{"assigned_to":%d}`, agent.ID))

	cases := []struct {
		name, query string
		want        []string
	}{
		{"por status", "?status=resolved", []string{"Falha no login"}},
		{"por prioridade", "?priority=high", []string{"Falha no login"}},
		{"por responsavel", fmt.Sprintf("?assigned_to=%d", agent.ID), []string{"Email fora do ar"}},
		{"busca em titulo", "?q=email", []string{"Email fora do ar"}},
		{"busca em descricao", "?q=desc", []string{"Impressora", "Email fora do ar", "Falha no login"}},
		{"combinados sem resultado", "?status=resolved&priority=low", []string{}},
		{"q em branco nao filtra", "?q=%20%20", []string{"Impressora", "Email fora do ar", "Falha no login"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := e.listTitles(t, agentTok, tc.query)
			if len(got) != len(tc.want) {
				t.Fatalf("query %s: %v, esperado %v", tc.query, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("query %s: %v, esperado %v", tc.query, got, tc.want)
				}
			}
		})
	}
}

func TestListTickets_InvalidFilter(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)

	for _, query := range []string{"?status=bogus", "?priority=urgent", "?assigned_to=0"} {
		if w := do(t, e.r, http.MethodGet, "/tickets"+query, "", e.token(t, agent)); w.Code != http.StatusBadRequest {
			t.Fatalf("GET /tickets%s: status %d, esperado 400", query, w.Code)
		}
	}
}

func TestListTickets_FilterRespectsRBAC(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)

	e.createTicket(t, e.token(t, custA), cat.ID, "segredo do A")
	e.createTicket(t, e.token(t, custB), cat.ID, "ticket do B")

	// Filtro não amplia a visibilidade: B busca o ticket de A e não acha nada.
	if got := e.listTitles(t, e.token(t, custB), "?q=segredo"); len(got) != 0 {
		t.Fatalf("customer B viu tickets alheios via filtro: %v", got)
	}
	if got := e.listTitles(t, e.token(t, custB), "?q=do+B"); len(got) != 1 {
		t.Fatalf("customer B nao achou o proprio ticket: %v", got)
	}
}

func TestGetTicket_CustomerCannotSeeOthers(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	custA := e.seedUser(t, tn.ID, "a@acme.com", services.RoleCustomer)
	custB := e.seedUser(t, tn.ID, "b@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)

	id := e.createTicket(t, e.token(t, custA), cat.ID, "secreto")

	// Customer B recebe 404 (não revela existência — ADR-003).
	if w := do(t, e.r, http.MethodGet, fmt.Sprintf("/tickets/%d", id), "", e.token(t, custB)); w.Code != http.StatusNotFound {
		t.Fatalf("custB GET alheio: status %d, esperado 404", w.Code)
	}
	// Agent vê normalmente.
	if w := do(t, e.r, http.MethodGet, fmt.Sprintf("/tickets/%d", id), "", e.token(t, agent)); w.Code != http.StatusOK {
		t.Fatalf("agent GET: status %d, esperado 200", w.Code)
	}
}

func TestGetTicket_CrossTenant404(t *testing.T) {
	e := newEnv(t)
	tn1 := e.seedTenant(t, "T1", "t1")
	cat := e.seedCategory(t, tn1.ID, "Bugs")
	owner := e.seedUser(t, tn1.ID, "o@t1.com", services.RoleCustomer)
	id := e.createTicket(t, e.token(t, owner), cat.ID, "do-tenant-1")

	tn2 := e.seedTenant(t, "T2", "t2")
	intruder := e.seedUser(t, tn2.ID, "x@t2.com", services.RoleAgent)

	// Mesmo sendo agent, não enxerga ticket de outro tenant (ADR-001/003).
	if w := do(t, e.r, http.MethodGet, fmt.Sprintf("/tickets/%d", id), "", e.token(t, intruder)); w.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant: status %d, esperado 404", w.Code)
	}
}

func TestUpdateTicket_CustomerForbiddenFields(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	tok := e.token(t, customer)
	id := e.createTicket(t, tok, cat.ID, "meu")

	// Mudar status é proibido para customer.
	if w := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", id), `{"status":"resolved"}`, tok); w.Code != http.StatusForbidden {
		t.Fatalf("customer muda status: status %d, esperado 403", w.Code)
	}
	// Editar título do próprio ticket é permitido.
	w := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", id), `{"title":"novo titulo"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("customer edita titulo: status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	if decode(t, w)["title"] != "novo titulo" {
		t.Fatal("titulo nao foi atualizado")
	}
}

func TestUpdateTicket_AgentAssignsAndResolves(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	id := e.createTicket(t, e.token(t, customer), cat.ID, "atribuir")

	body := fmt.Sprintf(`{"status":"resolved","assigned_to":%d,"priority":"high"}`, agent.ID)
	w := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", id), body, e.token(t, agent))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	got := decode(t, w)
	if got["status"] != "resolved" || got["priority"] != "high" {
		t.Fatalf("campos nao atualizados: %v", got)
	}
	if int64(got["assigned_to"].(float64)) != agent.ID {
		t.Fatalf("assigned_to %v, esperado %d", got["assigned_to"], agent.ID)
	}
	if got["resolved_at"] == nil {
		t.Fatal("resolved_at deveria ser carimbado ao resolver")
	}
}

func TestUpdateTicket_InvalidAssignee(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)
	agent := e.seedUser(t, tn.ID, "agent@acme.com", services.RoleAgent)
	id := e.createTicket(t, e.token(t, customer), cat.ID, "x")

	w := do(t, e.r, http.MethodPatch, fmt.Sprintf("/tickets/%d", id), `{"assigned_to":999}`, e.token(t, agent))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, esperado 422", w.Code)
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "invalid_assignee" {
		t.Fatalf("code %v, esperado invalid_assignee", code)
	}
}

func TestCreateCategory_AdminOnly(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	admin := e.seedUser(t, tn.ID, "admin@acme.com", services.RoleAdmin)
	customer := e.seedUser(t, tn.ID, "c@acme.com", services.RoleCustomer)

	// Admin cria.
	if w := do(t, e.r, http.MethodPost, "/categories", `{"name":"Rede"}`, e.token(t, admin)); w.Code != http.StatusCreated {
		t.Fatalf("admin cria categoria: status %d, esperado 201", w.Code)
	}
	// Customer é barrado pelo RequireRole.
	if w := do(t, e.r, http.MethodPost, "/categories", `{"name":"Outra"}`, e.token(t, customer)); w.Code != http.StatusForbidden {
		t.Fatalf("customer cria categoria: status %d, esperado 403", w.Code)
	}
}
