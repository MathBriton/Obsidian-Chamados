package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/database/testdb"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

func newAuthService(t *testing.T) (*services.AuthService, *repositories.Store) {
	t.Helper()
	store := repositories.NewStore(testdb.New(t))
	tokens := auth.NewTokenManager("test-secret", 15*time.Minute)
	return services.NewAuthService(store, tokens, 7*24*time.Hour), store
}

func register(t *testing.T, s *services.AuthService) services.AuthResult {
	t.Helper()
	res, err := s.Register(context.Background(), services.RegisterInput{
		TenantName: "Acme", Slug: "acme", Name: "Admin", Email: "admin@acme.com", Password: "senha-super-secreta",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	return res
}

func TestRegister_Success(t *testing.T) {
	s, _ := newAuthService(t)
	res := register(t, s)

	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatal("tokens não deveriam ser vazios")
	}
	if res.User.Role != "admin" {
		t.Fatalf("role %q, esperado admin", res.User.Role)
	}
	if res.User.TenantID != res.Tenant.ID {
		t.Fatal("usuário deveria pertencer ao tenant criado")
	}
}

func TestRegister_DuplicateSlug(t *testing.T) {
	s, _ := newAuthService(t)
	register(t, s)

	_, err := s.Register(context.Background(), services.RegisterInput{
		TenantName: "Outra", Slug: "acme", Name: "X", Email: "x@x.com", Password: "outrasenha",
	})
	if !errors.Is(err, models.ErrSlugTaken) {
		t.Fatalf("esperado ErrSlugTaken, obtido %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	s, _ := newAuthService(t)
	register(t, s)

	res, err := s.Login(context.Background(), "acme", "admin@acme.com", "senha-super-secreta")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatal("tokens vazios após login")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	s, _ := newAuthService(t)
	register(t, s)

	_, err := s.Login(context.Background(), "acme", "admin@acme.com", "senha-errada")
	if !errors.Is(err, models.ErrInvalidCredentials) {
		t.Fatalf("esperado ErrInvalidCredentials, obtido %v", err)
	}
}

// TestLogin_UnknownTenantOrEmail garante que recurso ausente não vaza: vira
// credencial inválida (ADR-003).
func TestLogin_UnknownTenantOrEmail(t *testing.T) {
	s, _ := newAuthService(t)
	register(t, s)

	cases := []struct{ slug, email string }{
		{"inexistente", "admin@acme.com"},
		{"acme", "ninguem@acme.com"},
	}
	for _, c := range cases {
		_, err := s.Login(context.Background(), c.slug, c.email, "senha-super-secreta")
		if !errors.Is(err, models.ErrInvalidCredentials) {
			t.Fatalf("slug=%q email=%q: esperado ErrInvalidCredentials, obtido %v", c.slug, c.email, err)
		}
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	s, store := newAuthService(t)
	res := register(t, s)

	if err := store.Users.Deactivate(context.Background(), res.Tenant.ID, res.User.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	_, err := s.Login(context.Background(), "acme", "admin@acme.com", "senha-super-secreta")
	if !errors.Is(err, models.ErrInactiveUser) {
		t.Fatalf("esperado ErrInactiveUser, obtido %v", err)
	}
}

func TestRefresh_RotatesAndInvalidatesOld(t *testing.T) {
	s, _ := newAuthService(t)
	res := register(t, s)

	rotated, err := s.Refresh(context.Background(), res.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if rotated.RefreshToken == res.RefreshToken {
		t.Fatal("o refresh token deveria ter sido rotacionado")
	}
	if rotated.AccessToken == "" {
		t.Fatal("access token vazio após refresh")
	}

	// O token antigo foi revogado e não pode ser reutilizado.
	if _, err := s.Refresh(context.Background(), res.RefreshToken); !errors.Is(err, models.ErrInvalidToken) {
		t.Fatalf("reutilizar refresh antigo deveria dar ErrInvalidToken, obtido %v", err)
	}
	// O novo token funciona.
	if _, err := s.Refresh(context.Background(), rotated.RefreshToken); err != nil {
		t.Fatalf("novo refresh deveria funcionar: %v", err)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	s, _ := newAuthService(t)
	register(t, s)

	if _, err := s.Refresh(context.Background(), "token-que-nao-existe"); !errors.Is(err, models.ErrInvalidToken) {
		t.Fatalf("esperado ErrInvalidToken, obtido %v", err)
	}
}

func TestLogout_RevokesRefresh(t *testing.T) {
	s, _ := newAuthService(t)
	res := register(t, s)

	if err := s.Logout(context.Background(), res.RefreshToken); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := s.Refresh(context.Background(), res.RefreshToken); !errors.Is(err, models.ErrInvalidToken) {
		t.Fatalf("refresh após logout deveria dar ErrInvalidToken, obtido %v", err)
	}
}

// TestAccessTokenCarriesClaims confirma que o access token emitido carrega
// tenant_id e role corretos (RNF01).
func TestAccessTokenCarriesClaims(t *testing.T) {
	s, _ := newAuthService(t)
	res := register(t, s)

	tm := auth.NewTokenManager("test-secret", 15*time.Minute)
	claims, err := tm.ParseAccessToken(res.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if claims.TenantID != res.Tenant.ID || claims.UserID != res.User.ID || claims.Role != "admin" {
		t.Fatalf("claims inesperados: %+v", claims)
	}
}
