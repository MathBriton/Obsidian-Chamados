package services_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// nt é um atalho para um sql.NullTime válido.
func nt(t time.Time) sql.NullTime { return sql.NullTime{Time: t, Valid: true} }

func TestSLAState(t *testing.T) {
	base := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	due := base.Add(100 * time.Minute) // janela de 100 min
	var null sql.NullTime

	cases := []struct {
		name          string
		dueAt, doneAt sql.NullTime
		now           time.Time
		want          string
	}{
		{"sem politica", null, null, base, services.SLAStateNone},
		{"cumprido no prazo", nt(due), nt(due.Add(-1 * time.Minute)), due, services.SLAStateMet},
		{"cumprido exatamente no prazo", nt(due), nt(due), due, services.SLAStateMet},
		{"cumprido apos o prazo", nt(due), nt(due.Add(time.Minute)), due, services.SLAStateBreached},
		{"pendente e dentro do prazo", nt(due), null, base.Add(50 * time.Minute), services.SLAStateOk},
		{"pendente no limiar de risco (20%)", nt(due), null, base.Add(80 * time.Minute), services.SLAStateAtRisk},
		{"pendente em risco", nt(due), null, base.Add(95 * time.Minute), services.SLAStateAtRisk},
		{"pendente e vencido", nt(due), null, due.Add(time.Minute), services.SLAStateBreached},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := services.SLAState(tc.dueAt, tc.doneAt, base, tc.now); got != tc.want {
				t.Fatalf("SLAState = %q, esperado %q", got, tc.want)
			}
		})
	}
}

// slaEnv prepara um TicketService sobre um tenant registrado (com as políticas
// de SLA padrão semeadas) e uma categoria pronta para abrir tickets.
type slaEnv struct {
	store    *repositories.Store
	tickets  *services.TicketService
	admin    services.Actor
	customer services.Actor
	category int64
}

func newSLAEnv(t *testing.T) slaEnv {
	t.Helper()
	authSvc, store := newAuthService(t)
	res := register(t, authSvc)

	cat, err := store.Categories.Create(context.Background(), res.Tenant.ID, "Bugs")
	if err != nil {
		t.Fatalf("seed category: %v", err)
	}
	cust, err := store.Users.Create(context.Background(), repositories.CreateUserInput{
		TenantID: res.Tenant.ID, Name: "Cli", Email: "cli@acme.com", PasswordHash: "x", Role: services.RoleCustomer,
	})
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	return slaEnv{
		store:    store,
		tickets:  services.NewTicketService(store),
		admin:    services.Actor{UserID: res.User.ID, TenantID: res.Tenant.ID, Role: services.RoleAdmin},
		customer: services.Actor{UserID: cust.ID, TenantID: res.Tenant.ID, Role: services.RoleCustomer},
		category: cat.ID,
	}
}

func TestRegister_SeedsDefaultSLAPolicies(t *testing.T) {
	authSvc, store := newAuthService(t)
	res := register(t, authSvc)

	policies, err := store.SLA.ListByTenant(context.Background(), res.Tenant.ID)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(policies) != len(services.DefaultSLAPolicies) {
		t.Fatalf("esperado %d políticas, obtido %d", len(services.DefaultSLAPolicies), len(policies))
	}
	// Ordenação por severidade: critical primeiro.
	if policies[0].Priority != "critical" {
		t.Fatalf("primeira política %q, esperado critical", policies[0].Priority)
	}
}

func TestCreateTicket_ComputesSLADueDates(t *testing.T) {
	e := newSLAEnv(t)
	ticket, err := e.tickets.Create(context.Background(), e.customer, services.CreateTicketInput{
		Title: "X", Description: "y", Priority: "high", CategoryID: e.category,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !ticket.FirstResponseDueAt.Valid || !ticket.ResolutionDueAt.Valid {
		t.Fatal("prazos de SLA deveriam estar preenchidos para prioridade com política")
	}
	// high = 30 min resposta / 240 min resolução; resolução vence depois.
	if !ticket.ResolutionDueAt.Time.After(ticket.FirstResponseDueAt.Time) {
		t.Fatal("prazo de resolução deveria ser posterior ao de primeira resposta")
	}
}

func TestUpdateTicket_PriorityChangeRecomputesSLA(t *testing.T) {
	e := newSLAEnv(t)
	ticket, err := e.tickets.Create(context.Background(), e.customer, services.CreateTicketInput{
		Title: "X", Description: "y", Priority: "low", CategoryID: e.category,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	lowDue := ticket.ResolutionDueAt.Time

	crit := "critical"
	updated, err := e.tickets.Update(context.Background(), e.admin, ticket.ID, services.UpdateTicketInput{Priority: &crit})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	// critical (120 min) tem janela bem menor que low (1440 min): vence antes.
	if !updated.ResolutionDueAt.Time.Before(lowDue) {
		t.Fatalf("prazo de resolução deveria encurtar ao subir a prioridade (%v >= %v)", updated.ResolutionDueAt.Time, lowDue)
	}
}

func TestAddComment_FirstResponseStamping(t *testing.T) {
	e := newSLAEnv(t)
	ctx := context.Background()
	ticket, err := e.tickets.Create(ctx, e.customer, services.CreateTicketInput{
		Title: "X", Description: "y", Priority: "medium", CategoryID: e.category,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	mustRespondedAt := func(label string) sql.NullTime {
		got, err := e.store.Tickets.GetByID(ctx, ticket.TenantID, ticket.ID)
		if err != nil {
			t.Fatalf("%s GetByID: %v", label, err)
		}
		return got.FirstRespondedAt
	}

	if mustRespondedAt("inicial").Valid {
		t.Fatal("first_responded_at deveria começar nulo")
	}

	// Comentário do customer não conta como primeira resposta.
	if _, err := e.tickets.AddComment(ctx, e.customer, ticket.ID, "oi", false); err != nil {
		t.Fatalf("AddComment customer: %v", err)
	}
	if mustRespondedAt("após customer").Valid {
		t.Fatal("comentário de customer não deveria carimbar a primeira resposta")
	}

	// Nota interna de staff também não conta.
	if _, err := e.tickets.AddComment(ctx, e.admin, ticket.ID, "nota", true); err != nil {
		t.Fatalf("AddComment nota interna: %v", err)
	}
	if mustRespondedAt("após nota interna").Valid {
		t.Fatal("nota interna não deveria carimbar a primeira resposta")
	}

	// Comentário público de staff carimba.
	if _, err := e.tickets.AddComment(ctx, e.admin, ticket.ID, "resposta", false); err != nil {
		t.Fatalf("AddComment público: %v", err)
	}
	stamped := mustRespondedAt("após resposta pública")
	if !stamped.Valid {
		t.Fatal("comentário público de staff deveria carimbar a primeira resposta")
	}

	// Segundo comentário público não altera o instante já registrado.
	if _, err := e.tickets.AddComment(ctx, e.admin, ticket.ID, "de novo", false); err != nil {
		t.Fatalf("AddComment segundo: %v", err)
	}
	if again := mustRespondedAt("após segundo"); !again.Time.Equal(stamped.Time) {
		t.Fatalf("segunda resposta moveu o carimbo: %v != %v", again.Time, stamped.Time)
	}
}

func TestListTickets_BreachedFilter(t *testing.T) {
	e := newSLAEnv(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Ticket estourado: prazo de resolução no passado, sem resolved_at.
	breached, err := e.store.Tickets.Create(ctx, repositories.CreateTicketInput{
		TenantID: e.admin.TenantID, Title: "tarde", Description: "d", Status: "open", Priority: "high",
		CategoryID: e.category, CreatedBy: e.admin.UserID,
		ResolutionDueAt: sql.NullTime{Time: now.Add(-time.Hour), Valid: true},
	})
	if err != nil {
		t.Fatalf("create breached: %v", err)
	}
	// Ticket no prazo: vence no futuro.
	if _, err := e.store.Tickets.Create(ctx, repositories.CreateTicketInput{
		TenantID: e.admin.TenantID, Title: "ok", Description: "d", Status: "open", Priority: "high",
		CategoryID: e.category, CreatedBy: e.admin.UserID,
		ResolutionDueAt: sql.NullTime{Time: now.Add(time.Hour), Valid: true},
	}); err != nil {
		t.Fatalf("create ok: %v", err)
	}

	got, err := e.store.Tickets.ListByTenant(ctx, e.admin.TenantID,
		repositories.TicketFilter{BreachedBefore: &now}, 50, 0)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(got) != 1 || got[0].ID != breached.ID {
		t.Fatalf("filtro breached deveria retornar só o ticket estourado, obtido %d itens", len(got))
	}
}
