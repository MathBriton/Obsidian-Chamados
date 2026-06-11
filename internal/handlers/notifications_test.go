package handlers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// seedNotificationForCustomer abre um ticket pelo customer e faz o admin
// comentar publicamente, gerando uma notificação para o customer. Devolve o
// token do customer.
func (e env) seedNotificationForCustomer(t *testing.T, tenantID int64, catID int64) string {
	t.Helper()
	customer := e.seedUser(t, tenantID, "c@acme.com", services.RoleCustomer)
	admin := e.seedUser(t, tenantID, "admin@acme.com", services.RoleAdmin)
	ctok := e.token(t, customer)
	id := e.createTicket(t, ctok, catID, "Falha")
	w := do(t, e.r, http.MethodPost, "/tickets/"+strconv.FormatInt(id, 10)+"/comments",
		`{"body":"estou vendo isso"}`, e.token(t, admin))
	if w.Code != http.StatusCreated {
		t.Fatalf("comentário admin: status %d, body=%s", w.Code, w.Body.String())
	}
	return ctok
}

func TestNotifications_ListAndUnreadCount(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	ctok := e.seedNotificationForCustomer(t, tn.ID, cat.ID)

	w := do(t, e.r, http.MethodGet, "/notifications/unread_count", "", ctok)
	if w.Code != http.StatusOK {
		t.Fatalf("unread_count status %d", w.Code)
	}
	if int64(decode(t, w)["count"].(float64)) != 1 {
		t.Fatalf("count esperado 1, body=%s", w.Body.String())
	}

	w = do(t, e.r, http.MethodGet, "/notifications?unread=true", "", ctok)
	if w.Code != http.StatusOK {
		t.Fatalf("list status %d", w.Code)
	}
	notifs := decode(t, w)["notifications"].([]interface{})
	if len(notifs) != 1 {
		t.Fatalf("esperado 1 notificação, obtido %d", len(notifs))
	}
	if notifs[0].(map[string]interface{})["kind"] != "comment" {
		t.Fatalf("kind inesperado: %v", notifs[0])
	}
}

func TestNotifications_MarkReadAndReadAll(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	ctok := e.seedNotificationForCustomer(t, tn.ID, cat.ID)

	// Descobre o id da notificação.
	w := do(t, e.r, http.MethodGet, "/notifications", "", ctok)
	notifs := decode(t, w)["notifications"].([]interface{})
	id := int64(notifs[0].(map[string]interface{})["id"].(float64))

	// Marca uma como lida.
	w = do(t, e.r, http.MethodPost, "/notifications/"+strconv.FormatInt(id, 10)+"/read", "", ctok)
	if w.Code != http.StatusNoContent {
		t.Fatalf("mark read status %d, esperado 204; body=%s", w.Code, w.Body.String())
	}
	w = do(t, e.r, http.MethodGet, "/notifications/unread_count", "", ctok)
	if int64(decode(t, w)["count"].(float64)) != 0 {
		t.Fatalf("após marcar lida, count deveria ser 0")
	}

	// read_all em estado já zerado é idempotente (204).
	w = do(t, e.r, http.MethodPost, "/notifications/read_all", "", ctok)
	if w.Code != http.StatusNoContent {
		t.Fatalf("read_all status %d, esperado 204", w.Code)
	}
}

func TestNotifications_MarkReadForeignIs404(t *testing.T) {
	e := newEnv(t)
	tn := e.seedTenant(t, "Acme", "acme")
	cat := e.seedCategory(t, tn.ID, "Bugs")
	e.seedNotificationForCustomer(t, tn.ID, cat.ID)

	// Outro usuário (agente) tenta marcar a notificação alheia (id 1): 404.
	agent := e.seedUser(t, tn.ID, "ag@acme.com", services.RoleAgent)
	w := do(t, e.r, http.MethodPost, "/notifications/1/read", "", e.token(t, agent))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d, esperado 404 ao marcar notificação alheia", w.Code)
	}
}
