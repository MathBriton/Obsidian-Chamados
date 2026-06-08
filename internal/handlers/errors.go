// Package handlers expõe o domínio sobre HTTP (Gin). Traduz entradas JSON em
// chamadas de service e mapeia erros de domínio para respostas estáveis (RNF13).
package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/models"
)

// errorBody é o corpo padronizado de erro da API. O campo Code é estável e
// destinado a clientes; Message é legível para humanos (RNF13).
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// respondError escreve um erro no formato {"error": {code, message}}.
func respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": errorBody{Code: code, Message: message}})
}

// respondDomainError mapeia um erro de domínio para o status e código HTTP
// correspondentes. Erros desconhecidos viram 500 sem vazar detalhes internos.
func respondDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrSlugTaken):
		respondError(c, http.StatusConflict, "slug_taken", err.Error())
	case errors.Is(err, models.ErrEmailTaken):
		respondError(c, http.StatusConflict, "email_taken", err.Error())
	case errors.Is(err, models.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, models.ErrInvalidToken):
		respondError(c, http.StatusUnauthorized, "invalid_token", err.Error())
	case errors.Is(err, models.ErrExpiredToken):
		respondError(c, http.StatusUnauthorized, "token_expired", err.Error())
	case errors.Is(err, models.ErrInactiveUser):
		respondError(c, http.StatusForbidden, "inactive_user", err.Error())
	case errors.Is(err, models.ErrForbidden):
		respondError(c, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, models.ErrNotFound):
		respondError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		respondError(c, http.StatusInternalServerError, "internal_error", "erro interno")
	}
}
