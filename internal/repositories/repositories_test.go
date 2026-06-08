package repositories_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/database/testdb"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// newStore monta um Store sobre um SQLite em memória limpo.
func newStore(t *testing.T) *repositories.Store {
	t.Helper()
	return repositories.NewStore(testdb.New(t))
}

func TestTenantRepository_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	created, err := s.Tenants.Create(ctx, "Acme", "acme")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.Slug != "acme" {
		t.Fatalf("tenant inesperado: %+v", created)
	}

	bySlug, err := s.Tenants.GetBySlug(ctx, "acme")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if bySlug.ID != created.ID {
		t.Fatalf("GetBySlug retornou id %d, esperado %d", bySlug.ID, created.ID)
	}

	byID, err := s.Tenants.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Slug != "acme" {
		t.Fatalf("GetByID slug %q", byID.Slug)
	}
}

func TestTenantRepository_DuplicateSlug(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	if _, err := s.Tenants.Create(ctx, "Acme", "acme"); err != nil {
		t.Fatalf("primeiro Create: %v", err)
	}
	_, err := s.Tenants.Create(ctx, "Outra", "acme")
	if !errors.Is(err, models.ErrSlugTaken) {
		t.Fatalf("esperado ErrSlugTaken, obtido %v", err)
	}
}

func TestTenantRepository_GetBySlug_NotFound(t *testing.T) {
	s := newStore(t)
	_, err := s.Tenants.GetBySlug(context.Background(), "inexistente")
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("esperado ErrNotFound, obtido %v", err)
	}
}

func TestUserRepository_CreateAndDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tenant, _ := s.Tenants.Create(ctx, "Acme", "acme")

	in := repositories.CreateUserInput{
		TenantID:     tenant.ID,
		Name:         "Admin",
		Email:        "admin@acme.com",
		PasswordHash: "hash",
		Role:         "admin",
	}
	user, err := s.Users.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.TenantID != tenant.ID {
		t.Fatalf("tenant_id %d, esperado %d", user.TenantID, tenant.ID)
	}

	_, err = s.Users.Create(ctx, in)
	if !errors.Is(err, models.ErrEmailTaken) {
		t.Fatalf("esperado ErrEmailTaken, obtido %v", err)
	}
}

// TestUserRepository_TenantIsolation cobre o ADR-002: o mesmo email pode
// existir em tenants diferentes e uma busca jamais cruza o tenant.
func TestUserRepository_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	acme, _ := s.Tenants.Create(ctx, "Acme", "acme")
	globex, _ := s.Tenants.Create(ctx, "Globex", "globex")

	mkUser := func(tenantID int64) {
		_, err := s.Users.Create(ctx, repositories.CreateUserInput{
			TenantID: tenantID, Name: "User", Email: "user@x.com", PasswordHash: "h", Role: "agent",
		})
		if err != nil {
			t.Fatalf("Create no tenant %d: %v", tenantID, err)
		}
	}
	mkUser(acme.ID)
	mkUser(globex.ID) // mesmo email, tenant diferente — deve ser permitido

	acmeUser, err := s.Users.GetByEmail(ctx, acme.ID, "user@x.com")
	if err != nil {
		t.Fatalf("GetByEmail acme: %v", err)
	}

	// Buscar o usuário do acme pelo id, mas escopado no globex, não encontra.
	_, err = s.Users.GetByID(ctx, globex.ID, acmeUser.ID)
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("cross-tenant GetByID deveria dar ErrNotFound, obtido %v", err)
	}
}

func TestUserRepository_Deactivate(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tenant, _ := s.Tenants.Create(ctx, "Acme", "acme")
	user, _ := s.Users.Create(ctx, repositories.CreateUserInput{
		TenantID: tenant.ID, Name: "A", Email: "a@a.com", PasswordHash: "h", Role: "agent",
	})

	if err := s.Users.Deactivate(ctx, tenant.ID, user.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	got, _ := s.Users.GetByID(ctx, tenant.ID, user.ID)
	if got.IsActive {
		t.Fatal("usuário deveria estar inativo")
	}
}

func TestRefreshTokenRepository_Lifecycle(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tenant, _ := s.Tenants.Create(ctx, "Acme", "acme")
	user, _ := s.Users.Create(ctx, repositories.CreateUserInput{
		TenantID: tenant.ID, Name: "A", Email: "a@a.com", PasswordHash: "h", Role: "agent",
	})

	exp := time.Now().Add(time.Hour).UTC()
	if _, err := s.Tokens.Create(ctx, tenant.ID, user.ID, "hash-1", exp); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Tokens.GetByHash(ctx, "hash-1")
	if err != nil {
		t.Fatalf("GetByHash: %v", err)
	}
	if got.RevokedAt.Valid {
		t.Fatal("token recém-criado não deveria estar revogado")
	}

	if err := s.Tokens.Revoke(ctx, "hash-1"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	got, _ = s.Tokens.GetByHash(ctx, "hash-1")
	if !got.RevokedAt.Valid {
		t.Fatal("token deveria estar revogado")
	}
}

// TestExecTx_Rollback garante que um erro dentro da transação desfaz tudo.
func TestExecTx_Rollback(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	sentinel := errors.New("boom")
	err := s.ExecTx(ctx, func(r *repositories.Repositories) error {
		if _, err := r.Tenants.Create(ctx, "Acme", "acme"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("esperado sentinel, obtido %v", err)
	}

	// O tenant criado dentro da tx deve ter sumido após o rollback.
	if _, err := s.Tenants.GetBySlug(ctx, "acme"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("rollback falhou: tenant ainda existe (%v)", err)
	}
}

// TestExecTx_Commit garante a persistência ao retornar nil.
func TestExecTx_Commit(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	err := s.ExecTx(ctx, func(r *repositories.Repositories) error {
		_, err := r.Tenants.Create(ctx, "Acme", "acme")
		return err
	})
	if err != nil {
		t.Fatalf("ExecTx: %v", err)
	}
	if _, err := s.Tenants.GetBySlug(ctx, "acme"); err != nil {
		t.Fatalf("commit falhou: %v", err)
	}
}
