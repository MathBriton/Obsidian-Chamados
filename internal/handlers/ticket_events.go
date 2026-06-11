package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/repositories"
)

// ticketEventResponse é um item do histórico do ticket. old_value/new_value
// são snapshots legíveis do momento do evento (enum de status/prioridade ou
// nome de categoria/usuário/equipe); null quando não se aplicam.
type ticketEventResponse struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind" example:"status_changed"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	ActorID   int64     `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	CreatedAt time.Time `json:"created_at"`
}

func toTicketEventResponse(e repositories.EventWithActor) ticketEventResponse {
	out := ticketEventResponse{
		ID:        e.ID,
		Kind:      e.Kind,
		ActorID:   e.ActorID,
		ActorName: e.ActorName,
		CreatedAt: e.CreatedAt,
	}
	if e.OldValue.Valid {
		out.OldValue = &e.OldValue.String
	}
	if e.NewValue.Valid {
		out.NewValue = &e.NewValue.String
	}
	return out
}

// ListTicketEvents devolve o histórico (auditoria) de um ticket, na mesma
// visibilidade do GET do ticket.
//
// @Summary   Histórico de um ticket
// @Tags      tickets
// @Produce   json
// @Security  Bearer
// @Param     id   path      int  true  "ID do ticket"
// @Success   200  {object}  ticketEventListResponse
// @Failure   401  {object}  errorEnvelope
// @Failure   404  {object}  errorEnvelope
// @Router    /tickets/{id}/events [get]
func (h *Handler) ListTicketEvents(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	events, err := h.tickets.ListEvents(c.Request.Context(), actor(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]ticketEventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, toTicketEventResponse(e))
	}
	c.JSON(http.StatusOK, gin.H{"events": out})
}
