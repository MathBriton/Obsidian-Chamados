package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/middleware"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

type registerRequest struct {
	TenantName string `json:"tenant_name" binding:"required"`
	Slug       string `json:"slug" binding:"required,lowercase,alphanum"`
	Name       string `json:"name" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	Slug     string `json:"slug" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ---------------------------------------------------------------------------
// Responses — nunca expõem password_hash nem timestamps internos.
// ---------------------------------------------------------------------------

type userResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type tenantResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type authResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	User         userResponse   `json:"user"`
	Tenant       tenantResponse `json:"tenant"`
}

func toUserResponse(u db.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role}
}

func toAuthResponse(r services.AuthResult) authResponse {
	return authResponse{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		User:         toUserResponse(r.User),
		Tenant:       tenantResponse{ID: r.Tenant.ID, Name: r.Tenant.Name, Slug: r.Tenant.Slug},
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// Register cria um tenant e seu usuário admin, devolvendo o par de tokens.
func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	res, err := h.auth.Register(c.Request.Context(), services.RegisterInput{
		TenantName: req.TenantName,
		Slug:       req.Slug,
		Name:       req.Name,
		Email:      req.Email,
		Password:   req.Password,
	})
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAuthResponse(res))
}

// Login autentica um usuário dentro do tenant identificado pelo slug.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	res, err := h.auth.Login(c.Request.Context(), req.Slug, req.Email, req.Password)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAuthResponse(res))
}

// Refresh rotaciona o refresh token e emite um novo par.
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	res, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toAuthResponse(res))
}

// Logout revoga o refresh token informado. Idempotente: responde 204 mesmo se
// o token já estava revogado ou não existe.
func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Me devolve o usuário autenticado, identificado pelos claims do access token.
func (h *Handler) Me(c *gin.Context) {
	user, err := h.auth.CurrentUser(c.Request.Context(), middleware.TenantID(c), middleware.UserID(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(user)})
}
