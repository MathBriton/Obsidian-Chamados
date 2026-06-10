package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/middleware"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

type createUserRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required,oneof=admin agent customer"`
}

// userAdminResponse expõe, para a gestão de usuários, campos além do /me
// (status ativo e data de criação). Nunca inclui password_hash.
type userAdminResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type userListResponse struct {
	Users []userAdminResponse `json:"users"`
}

// assignableUser é a visão mínima de um usuário para popular o select de
// responsável (não expõe email nem datas).
type assignableUser struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type assignableListResponse struct {
	Users []assignableUser `json:"users"`
}

func toUserAdminResponse(u db.User) userAdminResponse {
	return userAdminResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

// CreateUser cria um usuário no tenant (rota restrita a admin).
//
// @Summary   Cria um usuário (admin)
// @Tags      users
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     body  body      createUserRequest  true  "Dados do usuário"
// @Success   201   {object}  userAdminResponse
// @Failure   400   {object}  errorEnvelope
// @Failure   403   {object}  errorEnvelope
// @Failure   409   {object}  errorEnvelope
// @Router    /users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, err := h.users.Create(c.Request.Context(), middleware.TenantID(c), services.CreateUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserAdminResponse(user))
}

// ListUsers lista os usuários do tenant (rota restrita a admin).
//
// @Summary   Lista usuários do tenant (admin)
// @Tags      users
// @Produce   json
// @Security  Bearer
// @Param     limit   query     int  false  "Tamanho da página (máx. 50)"
// @Param     offset  query     int  false  "Deslocamento"
// @Success   200     {object}  userListResponse
// @Failure   403     {object}  errorEnvelope
// @Router    /users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	limit, offset := parsePagination(c)
	users, err := h.users.List(c.Request.Context(), middleware.TenantID(c), limit, offset)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]userAdminResponse, 0, len(users))
	for _, u := range users {
		out = append(out, toUserAdminResponse(u))
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

// ListAssignees lista os usuários atribuíveis a tickets (staff ativo).
// Acessível a staff (admin/agent), pois agents também atribuem chamados.
//
// @Summary   Lista usuários atribuíveis a tickets (staff)
// @Tags      users
// @Produce   json
// @Security  Bearer
// @Success   200  {object}  assignableListResponse
// @Failure   403  {object}  errorEnvelope
// @Router    /assignees [get]
func (h *Handler) ListAssignees(c *gin.Context) {
	users, err := h.users.ListAssignable(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]assignableUser, 0, len(users))
	for _, u := range users {
		out = append(out, assignableUser{ID: u.ID, Name: u.Name, Role: u.Role})
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

// DeactivateUser desativa um usuário do tenant (rota restrita a admin).
//
// @Summary   Desativa um usuário (admin)
// @Tags      users
// @Security  Bearer
// @Param     id   path  int  true  "ID do usuário"
// @Success   204  "Sem conteúdo"
// @Failure   403  {object}  errorEnvelope
// @Failure   404  {object}  errorEnvelope
// @Router    /users/{id} [delete]
func (h *Handler) DeactivateUser(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	err := h.users.Deactivate(c.Request.Context(), middleware.TenantID(c), middleware.UserID(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
