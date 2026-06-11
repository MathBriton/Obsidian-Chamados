package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/database/testdb"
	"github.com/MathBriton/Obsidian-Chamados/internal/handlers"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func newRouter(t *testing.T) *gin.Engine {
	t.Helper()
	store := repositories.NewStore(testdb.New(t))
	tokens := auth.NewTokenManager("test-secret", 15*time.Minute)
	svc := services.NewAuthService(store, tokens, 7*24*time.Hour)
	cats := services.NewCategoryService(store)
	tickets := services.NewTicketService(store)
	users := services.NewUserService(store)
	teams := services.NewTeamService(store)
	return handlers.New(svc, cats, tickets, users, teams, tokens).Router()
}

// do envia uma requisição JSON e devolve o response recorder.
func do(t *testing.T, r *gin.Engine, method, path, body, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return out
}

const validRegister = `{"tenant_name":"Acme","slug":"acme","name":"Admin","email":"admin@acme.com","password":"senha-super-secreta"}`

func TestRegister_Success(t *testing.T) {
	r := newRouter(t)
	w := do(t, r, http.MethodPost, "/auth/register", validRegister, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado 201; body=%s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["access_token"] == "" || body["refresh_token"] == "" {
		t.Fatal("tokens vazios na resposta")
	}
	// password_hash NUNCA pode vazar na resposta.
	if strings.Contains(w.Body.String(), "password") {
		t.Fatalf("resposta vazou campo de senha: %s", w.Body.String())
	}
	user := body["user"].(map[string]any)
	if user["role"] != "admin" {
		t.Fatalf("role %v, esperado admin", user["role"])
	}
}

func TestRegister_DuplicateSlug(t *testing.T) {
	r := newRouter(t)
	do(t, r, http.MethodPost, "/auth/register", validRegister, "")

	dup := `{"tenant_name":"Outra","slug":"acme","name":"X","email":"x@x.com","password":"outrasenha"}`
	w := do(t, r, http.MethodPost, "/auth/register", dup, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("status %d, esperado 409", w.Code)
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "slug_taken" {
		t.Fatalf("code %v, esperado slug_taken", code)
	}
}

func TestRegister_InvalidBody(t *testing.T) {
	r := newRouter(t)
	cases := map[string]string{
		"email inválido": `{"tenant_name":"A","slug":"acme","name":"N","email":"naoeemail","password":"senha-super"}`,
		"senha curta":    `{"tenant_name":"A","slug":"acme","name":"N","email":"a@a.com","password":"123"}`,
		"slug vazio":     `{"tenant_name":"A","slug":"","name":"N","email":"a@a.com","password":"senha-super"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			w := do(t, r, http.MethodPost, "/auth/register", body, "")
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status %d, esperado 400", w.Code)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	r := newRouter(t)
	do(t, r, http.MethodPost, "/auth/register", validRegister, "")

	login := `{"slug":"acme","email":"admin@acme.com","password":"senha-super-secreta"}`
	w := do(t, r, http.MethodPost, "/auth/login", login, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	if decode(t, w)["access_token"] == "" {
		t.Fatal("access_token vazio")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	r := newRouter(t)
	do(t, r, http.MethodPost, "/auth/register", validRegister, "")

	login := `{"slug":"acme","email":"admin@acme.com","password":"errada"}`
	w := do(t, r, http.MethodPost, "/auth/login", login, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, esperado 401", w.Code)
	}
	if code := decode(t, w)["error"].(map[string]any)["code"]; code != "invalid_credentials" {
		t.Fatalf("code %v, esperado invalid_credentials", code)
	}
}

func TestRefresh_RotatesAndInvalidatesOld(t *testing.T) {
	r := newRouter(t)
	reg := decode(t, do(t, r, http.MethodPost, "/auth/register", validRegister, ""))
	oldRefresh := reg["refresh_token"].(string)

	w := do(t, r, http.MethodPost, "/auth/refresh", `{"refresh_token":"`+oldRefresh+`"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200", w.Code)
	}
	if decode(t, w)["refresh_token"] == oldRefresh {
		t.Fatal("refresh token não foi rotacionado")
	}

	// Reutilizar o refresh antigo deve falhar.
	reuse := do(t, r, http.MethodPost, "/auth/refresh", `{"refresh_token":"`+oldRefresh+`"}`, "")
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("reutilizar refresh: status %d, esperado 401", reuse.Code)
	}
}

func TestLogout_ThenRefreshFails(t *testing.T) {
	r := newRouter(t)
	reg := decode(t, do(t, r, http.MethodPost, "/auth/register", validRegister, ""))
	refresh := reg["refresh_token"].(string)

	w := do(t, r, http.MethodPost, "/auth/logout", `{"refresh_token":"`+refresh+`"}`, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: status %d, esperado 204", w.Code)
	}

	after := do(t, r, http.MethodPost, "/auth/refresh", `{"refresh_token":"`+refresh+`"}`, "")
	if after.Code != http.StatusUnauthorized {
		t.Fatalf("refresh após logout: status %d, esperado 401", after.Code)
	}
}

func TestMe_RequiresValidToken(t *testing.T) {
	r := newRouter(t)
	reg := decode(t, do(t, r, http.MethodPost, "/auth/register", validRegister, ""))
	access := reg["access_token"].(string)

	// Sem token → 401.
	if w := do(t, r, http.MethodGet, "/me", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("/me sem token: status %d, esperado 401", w.Code)
	}

	// Token inválido → 401.
	if w := do(t, r, http.MethodGet, "/me", "", "lixo.invalido"); w.Code != http.StatusUnauthorized {
		t.Fatalf("/me token inválido: status %d, esperado 401", w.Code)
	}

	// Token válido → 200 com o usuário correto.
	w := do(t, r, http.MethodGet, "/me", "", access)
	if w.Code != http.StatusOK {
		t.Fatalf("/me com token: status %d, esperado 200; body=%s", w.Code, w.Body.String())
	}
	user := decode(t, w)["user"].(map[string]any)
	if user["email"] != "admin@acme.com" {
		t.Fatalf("email %v, esperado admin@acme.com", user["email"])
	}
}

func TestHealthz(t *testing.T) {
	r := newRouter(t)
	w := do(t, r, http.MethodGet, "/healthz", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, esperado 200", w.Code)
	}
	body := bytes.TrimSpace(w.Body.Bytes())
	if !bytes.Contains(body, []byte("ok")) {
		t.Fatalf("body inesperado: %s", body)
	}
}
