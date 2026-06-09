package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MathBriton/Obsidian-Chamados/internal/db"
	"github.com/MathBriton/Obsidian-Chamados/internal/middleware"
)

type createCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

type categoryResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func toCategoryResponse(c db.Category) categoryResponse {
	return categoryResponse{ID: c.ID, Name: c.Name}
}

// CreateCategory cria uma categoria no tenant (rota restrita a admin).
//
// @Summary   Cria uma categoria (admin)
// @Tags      categories
// @Accept    json
// @Produce   json
// @Security  Bearer
// @Param     body  body      createCategoryRequest  true  "Nome da categoria"
// @Success   201   {object}  categoryResponse
// @Failure   400   {object}  errorEnvelope
// @Failure   403   {object}  errorEnvelope
// @Failure   409   {object}  errorEnvelope
// @Router    /categories [post]
func (h *Handler) CreateCategory(c *gin.Context) {
	var req createCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	cat, err := h.categories.Create(c.Request.Context(), middleware.TenantID(c), req.Name)
	if err != nil {
		respondDomainError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toCategoryResponse(cat))
}

// ListCategories lista as categorias do tenant (qualquer papel autenticado).
//
// @Summary   Lista as categorias do tenant
// @Tags      categories
// @Produce   json
// @Security  Bearer
// @Success   200  {object}  categoryListResponse
// @Failure   401  {object}  errorEnvelope
// @Router    /categories [get]
func (h *Handler) ListCategories(c *gin.Context) {
	cats, err := h.categories.List(c.Request.Context(), middleware.TenantID(c))
	if err != nil {
		respondDomainError(c, err)
		return
	}
	out := make([]categoryResponse, 0, len(cats))
	for _, cat := range cats {
		out = append(out, toCategoryResponse(cat))
	}
	c.JSON(http.StatusOK, gin.H{"categories": out})
}
