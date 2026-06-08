package repositories

import (
	"context"
	"time"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

// RefreshTokenRepository dá acesso à tabela refresh_tokens. Armazena apenas
// o hash do token, nunca o valor em texto puro (RNF03).
type RefreshTokenRepository struct {
	q db.Querier
}

// Create persiste um refresh token (já hasheado) com seu vencimento.
func (r *RefreshTokenRepository) Create(ctx context.Context, tenantID, userID int64, tokenHash string, expiresAt time.Time) (db.RefreshToken, error) {
	return r.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		TenantID:  tenantID,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
}

// GetByHash busca um refresh token pelo hash. Retorna models.ErrNotFound se
// ausente.
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	token, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return db.RefreshToken{}, notFound(err)
	}
	return token, nil
}

// Revoke marca um único refresh token como revogado.
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	return r.q.RevokeRefreshToken(ctx, tokenHash)
}

// RevokeAll revoga todos os refresh tokens ativos de um usuário no tenant.
func (r *RefreshTokenRepository) RevokeAll(ctx context.Context, tenantID, userID int64) error {
	return r.q.RevokeAllUserTokens(ctx, db.RevokeAllUserTokensParams{TenantID: tenantID, UserID: userID})
}
