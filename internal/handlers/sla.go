package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/middleware"
)

// upsertSLAPolicyRequest são os tempos (em minutos) de uma política de SLA.
type upsertSLAPolicyRequest struct {
	FirstResponseMins int64 `json:"first_response_mins" binding:"required,gt=0"`
	ResolutionMins    int64 `json:"resolution_mins" binding:"required,gt=0"`
}

type slaPolicyResponse struct {
	Priority          string    `json:"priority"`
	FirstResponseMins int64     `json:"first_response_mins"`
	ResolutionMins    int64     `json:"resolution_mins"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func toSLAPolicyResponse(p db.SlaPolicy) slaPolicyResponse {
	return slaPolicyResponse{
		Priority:          p.Priority,
		FirstResponseMins: p.FirstResponseMins,
		ResolutionMins:    p.ResolutionMins,
		UpdatedAt:         p.UpdatedAt,
	}
}

// slaPriorities são as prioridades válidas para uma política de SLA.
var slaPriorities = map[string]bool{"low": true, "medium": true, "high": true, "critical": true}

// ListSLAPolicies devolve as políticas de SLA do tenant (staff).
//
// @Summary   Lista as políticas de SLA do tenant (staff)
// @Tags      sla
// @Produce   json
// @Security  Bearer
// @Success   200  {object}  slaPolicyListResponse
// @Failure   401  {object}  errorEnvelope
// @Failure   403  {object}  errorEnvelope
// @Router    /sla-policies [get]
func (h *Handler) ListSLAPolicies(c *gin.Context) {
	policies, err := h.sla.List(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]slaPolicyResponse, 0, len(policies))
	for _, p := range policies {
		out = append(out, toSLAPolicyResponse(p))
	}
	c.JSON(http.StatusOK, gin.H{"policies": out})
}

// UpsertSLAPolicy cria ou atualiza a política de SLA de uma prioridade (admin).
//
// @Summary   Define a política de SLA de uma prioridade (admin)
// @Tags      sla
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     priority  path      string                  true  "Prioridade"  Enums(low, medium, high, critical)
// @Param     body      body      upsertSLAPolicyRequest  true  "Tempos de SLA em minutos"
// @Success   200       {object}  slaPolicyResponse
// @Failure   400       {object}  errorEnvelope
// @Failure   403       {object}  errorEnvelope
// @Router    /sla-policies/{priority} [put]
func (h *Handler) UpsertSLAPolicy(c *gin.Context) {
	priority := c.Param("priority")
	if !slaPriorities[priority] {
		respondError(c, http.StatusBadRequest, "invalid_request", "prioridade inválida")
		return
	}
	var req upsertSLAPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	policy, err := h.sla.Upsert(c.Request.Context(), middleware.TenantID(c), priority, req.FirstResponseMins, req.ResolutionMins)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSLAPolicyResponse(policy))
}
