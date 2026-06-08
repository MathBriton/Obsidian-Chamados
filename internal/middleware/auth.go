// Package middleware concentra os middlewares HTTP transversais: autenticação
// via access token e propagação do tenant para as camadas seguintes (RNF01).
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
)

// Chaves usadas para guardar a identidade autenticada no contexto do Gin.
// O tenant_id vem SEMPRE do token assinado, nunca de um header ou body (RNF01).
const (
	ctxUserID   = "user_id"
	ctxTenantID = "tenant_id"
	ctxRole     = "role"
)

const bearerPrefix = "Bearer "

// RequireAuth exige um access token válido no header Authorization. Em caso de
// ausência, formato inválido ou assinatura/expiração inválidas, aborta com 401.
func RequireAuth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			unauthorized(c)
			return
		}

		claims, err := tokens.ParseAccessToken(strings.TrimPrefix(header, bearerPrefix))
		if err != nil {
			unauthorized(c)
			return
		}

		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxTenantID, claims.TenantID)
		c.Set(ctxRole, claims.Role)
		c.Next()
	}
}

// RequireRole exige que o usuário autenticado tenha um dos papéis informados.
// Deve ser encadeado após RequireAuth. Responde 403 caso contrário.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		if !allowed[Role(c)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "forbidden", "message": "acesso negado para o seu papel"},
			})
			return
		}
		c.Next()
	}
}

// UserID retorna o id do usuário autenticado; 0 se ausente.
func UserID(c *gin.Context) int64 { return int64Value(c, ctxUserID) }

// TenantID retorna o tenant do usuário autenticado; 0 se ausente.
func TenantID(c *gin.Context) int64 { return int64Value(c, ctxTenantID) }

// Role retorna o papel do usuário autenticado; "" se ausente.
func Role(c *gin.Context) string {
	v, _ := c.Get(ctxRole)
	role, _ := v.(string)
	return role
}

func int64Value(c *gin.Context, key string) int64 {
	v, _ := c.Get(key)
	id, _ := v.(int64)
	return id
}

func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{"code": "invalid_token", "message": "token de acesso ausente ou inválido"},
	})
}
