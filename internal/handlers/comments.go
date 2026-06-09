package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
)

type createCommentRequest struct {
	Body       string `json:"body" binding:"required,min=1"`
	IsInternal bool   `json:"is_internal"`
}

type commentResponse struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticket_id"`
	AuthorID   int64     `json:"author_id"`
	Body       string    `json:"body"`
	IsInternal bool      `json:"is_internal"`
	CreatedAt  time.Time `json:"created_at"`
}

func toCommentResponse(c db.Comment) commentResponse {
	return commentResponse{
		ID:         c.ID,
		TicketID:   c.TicketID,
		AuthorID:   c.AuthorID,
		Body:       c.Body,
		IsInternal: c.IsInternal,
		CreatedAt:  c.CreatedAt,
	}
}

// CreateComment adiciona um comentário a um ticket. Notas internas
// (is_internal) só são aceitas de staff.
//
// @Summary   Comenta em um ticket
// @Tags      comments
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     id    path      int                   true  "ID do ticket"
// @Param     body  body      createCommentRequest  true  "Comentário"
// @Success   201   {object}  commentResponse
// @Failure   400   {object}  errorEnvelope
// @Failure   403   {object}  errorEnvelope
// @Failure   404   {object}  errorEnvelope
// @Router    /tickets/{id}/comments [post]
func (h *Handler) CreateComment(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	comment, err := h.tickets.AddComment(c.Request.Context(), actor(c), id, req.Body, req.IsInternal)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toCommentResponse(comment))
}

// ListComments lista os comentários visíveis de um ticket (customers não veem
// notas internas).
//
// @Summary   Lista comentários de um ticket
// @Tags      comments
// @Produce   json
// @Security  Bearer
// @Param     id   path      int  true  "ID do ticket"
// @Success   200  {object}  commentListResponse
// @Failure   404  {object}  errorEnvelope
// @Router    /tickets/{id}/comments [get]
func (h *Handler) ListComments(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	comments, err := h.tickets.ListComments(c.Request.Context(), actor(c), id)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]commentResponse, 0, len(comments))
	for _, cm := range comments {
		out = append(out, toCommentResponse(cm))
	}
	c.JSON(http.StatusOK, gin.H{"comments": out})
}
