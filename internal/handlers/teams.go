package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/middleware"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

type createTeamRequest struct {
	Name string `json:"name" binding:"required"`
}

type addTeamMemberRequest struct {
	UserID int64 `json:"user_id" binding:"required,gt=0"`
}

type teamMemberResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type teamResponse struct {
	ID      int64                `json:"id"`
	Name    string               `json:"name"`
	Members []teamMemberResponse `json:"members"`
}

func toTeamResponse(t services.TeamWithMembers) teamResponse {
	members := make([]teamMemberResponse, 0, len(t.Members))
	for _, m := range t.Members {
		members = append(members, teamMemberResponse{ID: m.UserID, Name: m.Name, Role: m.Role})
	}
	return teamResponse{ID: t.Team.ID, Name: t.Team.Name, Members: members}
}

// CreateTeam cria uma equipe no tenant (rota restrita a admin).
//
// @Summary   Cria uma equipe (admin)
// @Tags      teams
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     body  body      createTeamRequest  true  "Nome da equipe"
// @Success   201   {object}  teamResponse
// @Failure   400   {object}  errorEnvelope
// @Failure   403   {object}  errorEnvelope
// @Failure   409   {object}  errorEnvelope
// @Router    /teams [post]
func (h *Handler) CreateTeam(c *gin.Context) {
	var req createTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	team, err := h.teams.Create(c.Request.Context(), middleware.TenantID(c), req.Name)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, teamResponse{ID: team.ID, Name: team.Name, Members: []teamMemberResponse{}})
}

// ListTeams lista as equipes do tenant com seus membros (staff).
//
// @Summary   Lista as equipes do tenant com membros (staff)
// @Tags      teams
// @Produce   json
// @Security  Bearer
// @Success   200  {object}  teamListResponse
// @Failure   401  {object}  errorEnvelope
// @Failure   403  {object}  errorEnvelope
// @Router    /teams [get]
func (h *Handler) ListTeams(c *gin.Context) {
	teams, err := h.teams.List(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]teamResponse, 0, len(teams))
	for _, t := range teams {
		out = append(out, toTeamResponse(t))
	}
	c.JSON(http.StatusOK, gin.H{"teams": out})
}

// AddTeamMember vincula um usuário staff à equipe (admin). Idempotente.
//
// @Summary   Adiciona um membro à equipe (admin)
// @Tags      teams
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     id    path      int                   true  "ID da equipe"
// @Param     body  body      addTeamMemberRequest  true  "Usuário a vincular"
// @Success   204   "membro vinculado"
// @Failure   400   {object}  errorEnvelope
// @Failure   403   {object}  errorEnvelope
// @Failure   404   {object}  errorEnvelope
// @Failure   422   {object}  errorEnvelope
// @Router    /teams/{id}/members [post]
func (h *Handler) AddTeamMember(c *gin.Context) {
	teamID, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req addTeamMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.teams.AddMember(c.Request.Context(), middleware.TenantID(c), teamID, req.UserID); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// RemoveTeamMember desvincula um usuário da equipe (admin). Idempotente.
//
// @Summary   Remove um membro da equipe (admin)
// @Tags      teams
// @Produce   json
// @Security  Bearer
// @Param     id      path  int  true  "ID da equipe"
// @Param     userID  path  int  true  "ID do usuário"
// @Success   204     "membro desvinculado"
// @Failure   400     {object}  errorEnvelope
// @Failure   403     {object}  errorEnvelope
// @Failure   404     {object}  errorEnvelope
// @Router    /teams/{id}/members/{userID} [delete]
func (h *Handler) RemoveTeamMember(c *gin.Context) {
	teamID, ok := parseIDParam(c)
	if !ok {
		return
	}
	userID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || userID <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "id de usuário inválido")
		return
	}

	if err := h.teams.RemoveMember(c.Request.Context(), middleware.TenantID(c), teamID, userID); err != nil {
		respondDomainError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
