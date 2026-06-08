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
	auth       *services.AuthService
	categories *services.CategoryService
	tickets    *services.TicketService
	tokens     *auth.TokenManager
}

// New monta o Handler a partir dos serviços de aplicação.
func New(
	authService *services.AuthService,
	categoryService *services.CategoryService,
	ticketService *services.TicketService,
	tokens *auth.TokenManager,
) *Handler {
	return &Handler{
		auth:       authService,
		categories: categoryService,
		tickets:    ticketService,
		tokens:     tokens,
	}
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

	// Rotas protegidas: exigem access token válido (RNF01).
	api := r.Group("/", middleware.RequireAuth(h.tokens))
	{
		api.GET("/me", h.Me)

		api.GET("/categories", h.ListCategories)
		// Apenas admin cria categorias.
		api.POST("/categories", middleware.RequireRole(services.RoleAdmin), h.CreateCategory)

		api.POST("/tickets", h.CreateTicket)
		api.GET("/tickets", h.ListTickets)
		api.GET("/tickets/:id", h.GetTicket)
		api.PATCH("/tickets/:id", h.UpdateTicket)
	}

	return r
}

// health é um liveness probe simples para orquestradores e o CI.
func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// actor monta a identidade autenticada a partir do contexto (claims do JWT).
func actor(c *gin.Context) services.Actor {
	return services.Actor{
		UserID:   middleware.UserID(c),
		TenantID: middleware.TenantID(c),
		Role:     middleware.Role(c),
	}
}
