package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// statsResponse resume os tickets visíveis ao usuário. Os mapas vêm
// zero-preenchidos com todos os valores dos enums, para a UI renderizar
// direto sem tratar chave ausente.
type statsResponse struct {
	Total            int64            `json:"total"`
	UnassignedActive int64            `json:"unassigned_active"`
	ByStatus         map[string]int64 `json:"by_status"`
	ByPriority       map[string]int64 `json:"by_priority"`
}

// GetStats devolve as contagens de tickets no escopo de visibilidade do
// usuário (RBAC): tenant inteiro para staff, só os próprios para customer.
//
// @Summary   Métricas de tickets (escopo do papel)
// @Tags      tickets
// @Produce   json
// @Security  Bearer
// @Success   200  {object}  statsResponse
// @Failure   401  {object}  errorEnvelope
// @Router    /stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.tickets.Stats(c.Request.Context(), actor(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}

	out := statsResponse{
		UnassignedActive: stats.UnassignedActive,
		ByStatus:         zeroFill(statusEnum, stats.ByStatus),
		ByPriority:       zeroFill(priorityEnum, stats.ByPriority),
	}
	for _, n := range out.ByStatus {
		out.Total += n
	}
	c.JSON(http.StatusOK, out)
}

// zeroFill projeta as contagens sobre todos os valores do enum, garantindo 0
// para os ausentes.
func zeroFill(enum string, counts map[string]int64) map[string]int64 {
	out := make(map[string]int64)
	for _, key := range strings.Fields(enum) {
		out[key] = counts[key]
	}
	return out
}
