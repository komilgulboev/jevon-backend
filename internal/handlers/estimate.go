package handlers

import (
	"net/http"
	"strconv"

	"jevon/internal/middleware"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type EstimateHandler struct {
	repo *repository.EstimateRepo
}

func NewEstimateHandler(repo *repository.EstimateRepo) *EstimateHandler {
	return &EstimateHandler{repo: repo}
}

// GET /api/estimate/catalog
func (h *EstimateHandler) CatalogList(c *gin.Context) {
	grouped, err := h.repo.CatalogList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":         grouped,
		"group_labels": repository.GroupLabels,
	})
}

// GET /api/estimate/catalog/flat
func (h *EstimateHandler) CatalogFlat(c *gin.Context) {
	items, err := h.repo.CatalogFlat(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil { items = []repository.ServiceCatalogItem{} }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// POST /api/estimate/catalog
func (h *EstimateHandler) CatalogCreate(c *gin.Context) {
	var req repository.ServiceCatalogItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.CatalogCreate(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// PATCH /api/estimate/catalog/:id
func (h *EstimateHandler) CatalogUpdate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req repository.ServiceCatalogItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.CatalogUpdate(c, id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DELETE /api/estimate/catalog/:id
func (h *EstimateHandler) CatalogDelete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.CatalogDelete(c, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GET /api/estimate/colors
func (h *EstimateHandler) ColorList(c *gin.Context) {
	colors, err := h.repo.ColorList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if colors == nil { colors = []repository.ColorItem{} }
	c.JSON(http.StatusOK, gin.H{"data": colors})
}

// GET /api/orders/:order_id/estimate
func (h *EstimateHandler) EstimateGet(c *gin.Context) {
	services, materials, totalSvc, totalMat, err := h.repo.EstimateByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if services  == nil { services  = []repository.EstimateServiceRow{}  }
	if materials == nil { materials = []repository.EstimateMaterialRow{} }
	c.JSON(http.StatusOK, gin.H{
		"services":        services,
		"materials":       materials,
		"total_services":  totalSvc,
		"total_materials": totalMat,
		"total":           totalSvc + totalMat,
	})
}

// POST /api/orders/:order_id/estimate
func (h *EstimateHandler) EstimateSave(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req repository.SaveEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SaveEstimate(c, c.Param("order_id"), claims.UserID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}
