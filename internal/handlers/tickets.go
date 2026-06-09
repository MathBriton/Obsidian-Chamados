package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/services"
)

const (
	statusEnum   = "open in_progress waiting_customer resolved closed"
	priorityEnum = "low medium high critical"
)

// ---------------------------------------------------------------------------
// Requests
// ---------------------------------------------------------------------------

type createTicketRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	CategoryID  int64  `json:"category_id" binding:"required"`
	Priority    string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
}

// updateTicketRequest usa ponteiros para distinguir campo ausente de valor
// zero numa atualização parcial (PATCH).
type updateTicketRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=1"`
	Description *string `json:"description" binding:"omitempty,min=1"`
	Status      *string `json:"status" binding:"omitempty,oneof=open in_progress waiting_customer resolved closed"`
	Priority    *string `json:"priority" binding:"omitempty,oneof=low medium high critical"`
	CategoryID  *int64  `json:"category_id" binding:"omitempty,gt=0"`
	AssignedTo  *int64  `json:"assigned_to" binding:"omitempty,gt=0"`
}

// ---------------------------------------------------------------------------
// Response
// ---------------------------------------------------------------------------

type ticketResponse struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	CategoryID     int64      `json:"category_id"`
	CreatedBy      int64      `json:"created_by"`
	AssignedTo     *int64     `json:"assigned_to"`
	AssignedTeamID *int64     `json:"assigned_team_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
	ClosedAt       *time.Time `json:"closed_at"`
}

func toTicketResponse(t db.Ticket) ticketResponse {
	return ticketResponse{
		ID:             t.ID,
		Title:          t.Title,
		Description:    t.Description,
		Status:         t.Status,
		Priority:       t.Priority,
		CategoryID:     t.CategoryID,
		CreatedBy:      t.CreatedBy,
		AssignedTo:     nullInt(t.AssignedTo),
		AssignedTeamID: nullInt(t.AssignedTeamID),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		ResolvedAt:     nullTime(t.ResolvedAt),
		ClosedAt:       nullTime(t.ClosedAt),
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// CreateTicket abre um ticket em nome do usuário autenticado.
//
// @Summary   Abre um ticket
// @Tags      tickets
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     body  body      createTicketRequest  true  "Dados do ticket"
// @Success   201   {object}  ticketResponse
// @Failure   400   {object}  errorEnvelope
// @Failure   422   {object}  errorEnvelope
// @Router    /tickets [post]
func (h *Handler) CreateTicket(c *gin.Context) {
	var req createTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ticket, err := h.tickets.Create(c.Request.Context(), actor(c), services.CreateTicketInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		CategoryID:  req.CategoryID,
	})
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTicketResponse(ticket))
}

// ListTickets lista os tickets visíveis ao usuário (RBAC no service).
//
// @Summary   Lista tickets (customer vê só os próprios)
// @Tags      tickets
// @Produce   json
// @Security  Bearer
// @Param     limit   query     int  false  "Tamanho da página (máx. 50)"
// @Param     offset  query     int  false  "Deslocamento"
// @Success   200     {object}  ticketListResponse
// @Failure   401     {object}  errorEnvelope
// @Router    /tickets [get]
func (h *Handler) ListTickets(c *gin.Context) {
	limit, offset := parsePagination(c)

	tickets, err := h.tickets.List(c.Request.Context(), actor(c), limit, offset)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]ticketResponse, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, toTicketResponse(t))
	}
	c.JSON(http.StatusOK, gin.H{"tickets": out})
}

// GetTicket devolve um ticket pelo id, respeitando a visibilidade do usuário.
//
// @Summary   Detalha um ticket
// @Tags      tickets
// @Produce   json
// @Security  Bearer
// @Param     id   path      int  true  "ID do ticket"
// @Success   200  {object}  ticketResponse
// @Failure   404  {object}  errorEnvelope
// @Router    /tickets/{id} [get]
func (h *Handler) GetTicket(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	ticket, err := h.tickets.Get(c.Request.Context(), actor(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTicketResponse(ticket))
}

// UpdateTicket aplica uma atualização parcial respeitando o RBAC.
//
// @Summary   Atualiza um ticket (parcial)
// @Description  Customer só edita título/descrição do próprio; agent/admin editam todos os campos.
// @Tags      tickets
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     id    path      int                  true  "ID do ticket"
// @Param     body  body      updateTicketRequest  true  "Campos a atualizar"
// @Success   200   {object}  ticketResponse
// @Failure   400   {object}  errorEnvelope
// @Failure   403   {object}  errorEnvelope
// @Failure   404   {object}  errorEnvelope
// @Failure   422   {object}  errorEnvelope
// @Router    /tickets/{id} [patch]
func (h *Handler) UpdateTicket(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req updateTicketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ticket, err := h.tickets.Update(c.Request.Context(), actor(c), id, services.UpdateTicketInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		CategoryID:  req.CategoryID,
		AssignedTo:  req.AssignedTo,
	})
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTicketResponse(ticket))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parsePagination lê os parâmetros de query limit/offset (0 quando ausentes;
// a normalização/limites ficam na camada de service).
func parsePagination(c *gin.Context) (limit, offset int64) {
	limit, _ = strconv.ParseInt(c.Query("limit"), 10, 64)
	offset, _ = strconv.ParseInt(c.Query("offset"), 10, 64)
	return limit, offset
}

// parseIDParam lê o :id da rota. Em caso de id inválido, responde 400 e
// retorna ok=false.
func parseIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "id inválido")
		return 0, false
	}
	return id, true
}

func nullInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

func nullTime(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}
