// Package services concentra as regras de negócio, orquestrando a camada de
// repositories e os utilitários de auth. Não conhece HTTP.
package services

import (
	"context"
	"errors"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/models"
	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// roleAdmin é o papel atribuído ao primeiro usuário de um tenant no registro.
const roleAdmin = "admin"

// AuthService implementa o fluxo de autenticação: registro de tenant,
// login, refresh com rotação e logout.
type AuthService struct {
	store      *repositories.Store
	tokens     *auth.TokenManager
	refreshTTL time.Duration
}

// NewAuthService monta o serviço de autenticação.
func NewAuthService(store *repositories.Store, tokens *auth.TokenManager, refreshTTL time.Duration) *AuthService {
	return &AuthService{store: store, tokens: tokens, refreshTTL: refreshTTL}
}

// RegisterInput agrega os dados do registro de um novo tenant + admin.
type RegisterInput struct {
	TenantName string
	Slug       string
	Name       string
	Email      string
	Password   string
}

// AuthResult é o retorno dos fluxos que autenticam um usuário.
type AuthResult struct {
	User         db.User
	Tenant       db.Tenant
	AccessToken  string
	RefreshToken string
}

// Register cria atomicamente um tenant e seu usuário admin, e já emite o par
// de tokens. Slug ou email duplicados retornam erros de domínio.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return AuthResult{}, err
	}

	rawRefresh, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}

	var tenant db.Tenant
	var user db.User
	err = s.store.ExecTx(ctx, func(r *repositories.Repositories) error {
		tenant, err = r.Tenants.Create(ctx, in.TenantName, in.Slug)
		if err != nil {
			return err
		}
		user, err = r.Users.Create(ctx, repositories.CreateUserInput{
			TenantID:     tenant.ID,
			Name:         in.Name,
			Email:        in.Email,
			PasswordHash: hash,
			Role:         roleAdmin,
		})
		if err != nil {
			return err
		}
		// Semeia as políticas de SLA padrão do tenant recém-criado.
		for _, p := range DefaultSLAPolicies {
			if _, err = r.SLA.Upsert(ctx, repositories.UpsertPolicyInput{
				TenantID:          tenant.ID,
				Priority:          p.Priority,
				FirstResponseMins: p.FirstResponseMins,
				ResolutionMins:    p.ResolutionMins,
			}); err != nil {
				return err
			}
		}
		_, err = r.Tokens.Create(ctx, tenant.ID, user.ID, refreshHash, s.refreshExpiry())
		return err
	})
	if err != nil {
		return AuthResult{}, err
	}

	access, err := s.tokens.GenerateAccessToken(user.ID, tenant.ID, user.Role)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Tenant: tenant, AccessToken: access, RefreshToken: rawRefresh}, nil
}

// Login autentica um usuário dentro do tenant identificado pelo slug.
// Para não revelar a existência de tenants ou emails (ADR-003), qualquer
// recurso ausente vira ErrInvalidCredentials.
func (s *AuthService) Login(ctx context.Context, slug, email, password string) (AuthResult, error) {
	tenant, err := s.store.Tenants.GetBySlug(ctx, slug)
	if err != nil {
		return AuthResult{}, hideNotFound(err, models.ErrInvalidCredentials)
	}

	user, err := s.store.Users.GetByEmail(ctx, tenant.ID, email)
	if err != nil {
		return AuthResult{}, hideNotFound(err, models.ErrInvalidCredentials)
	}

	if !user.IsActive {
		return AuthResult{}, models.ErrInactiveUser
	}
	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return AuthResult{}, models.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, tenant, user)
}

// Refresh valida um refresh token opaco, faz sua rotação (revoga o antigo e
// emite um novo par) e devolve tokens renovados.
func (s *AuthService) Refresh(ctx context.Context, rawRefresh string) (AuthResult, error) {
	hash := auth.HashRefreshToken(rawRefresh)
	row, err := s.store.Tokens.GetByHash(ctx, hash)
	if err != nil {
		return AuthResult{}, hideNotFound(err, models.ErrInvalidToken)
	}
	if row.RevokedAt.Valid {
		return AuthResult{}, models.ErrInvalidToken
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return AuthResult{}, models.ErrExpiredToken
	}

	user, err := s.store.Users.GetByID(ctx, row.TenantID, row.UserID)
	if err != nil {
		return AuthResult{}, hideNotFound(err, models.ErrInvalidToken)
	}
	if !user.IsActive {
		return AuthResult{}, models.ErrInactiveUser
	}
	tenant, err := s.store.Tenants.GetByID(ctx, row.TenantID)
	if err != nil {
		return AuthResult{}, err
	}

	// Rotação: o refresh usado é revogado antes de emitir o novo.
	if err := s.store.Tokens.Revoke(ctx, hash); err != nil {
		return AuthResult{}, err
	}
	return s.issueTokens(ctx, tenant, user)
}

// Logout revoga o refresh token informado. É idempotente: revogar um token
// inexistente ou já revogado não é erro.
func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	return s.store.Tokens.Revoke(ctx, auth.HashRefreshToken(rawRefresh))
}

// CurrentUser retorna o usuário autenticado a partir do tenant e id extraídos
// do access token. O filtro por tenant_id é reforçado no repositório (ADR-001).
func (s *AuthService) CurrentUser(ctx context.Context, tenantID, userID int64) (db.User, error) {
	return s.store.Users.GetByID(ctx, tenantID, userID)
}

// issueTokens emite um access token e um novo refresh token persistido.
func (s *AuthService) issueTokens(ctx context.Context, tenant db.Tenant, user db.User) (AuthResult, error) {
	access, err := s.tokens.GenerateAccessToken(user.ID, tenant.ID, user.Role)
	if err != nil {
		return AuthResult{}, err
	}
	rawRefresh, refreshHash, err := auth.GenerateRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}
	if _, err := s.store.Tokens.Create(ctx, tenant.ID, user.ID, refreshHash, s.refreshExpiry()); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, Tenant: tenant, AccessToken: access, RefreshToken: rawRefresh}, nil
}

func (s *AuthService) refreshExpiry() time.Time {
	return time.Now().UTC().Add(s.refreshTTL)
}

// hideNotFound substitui ErrNotFound por um erro genérico para não revelar a
// existência de recursos; demais erros passam inalterados.
func hideNotFound(err, replacement error) error {
	if errors.Is(err, models.ErrNotFound) {
		return replacement
	}
	return err
}
