package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/auth"
	"github.com/MathBriton/Obsidian-Chamados/internal/middleware"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// Handler agrega as dependências necessárias para servir as rotas HTTP.
type Handler struct {
	auth   *services.AuthService
	tokens *auth.TokenManager
}

// New monta o Handler a partir dos serviços de aplicação.
func New(authService *services.AuthService, tokens *auth.TokenManager) *Handler {
	return &Handler{auth: authService, tokens: tokens}
}

// Router constrói o *gin.Engine com todas as rotas registradas.
func (h *Handler) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", h.health)

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.Refresh)
		authGroup.POST("/logout", h.Logout)
	}

	// Rota protegida: exige access token válido (RNF01).
	r.GET("/me", middleware.RequireAuth(h.tokens), h.Me)

	return r
}

// health é um liveness probe simples para orquestradores e o CI.
func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
