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
