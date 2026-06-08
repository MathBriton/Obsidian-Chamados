package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims é o payload do access token JWT.
// tenant_id NUNCA vem do cliente — é gravado aqui na emissão e lido no middleware (RNF01).
type Claims struct {
	UserID   int64  `json:"user_id"`
	TenantID int64  `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager encapsula a emissão e validação de access tokens.
type TokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

// NewTokenManager cria um gerenciador de tokens.
// accessTTL deve ser 15min conforme RNF04.
func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:    []byte(secret),
		accessTTL: accessTTL,
	}
}

// GenerateAccessToken emite um access token assinado para o usuário.
func (m *TokenManager) GenerateAccessToken(userID, tenantID int64, role string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseAccessToken valida a assinatura e a expiração, retornando os claims.
func (m *TokenManager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de assinatura inesperado")
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
