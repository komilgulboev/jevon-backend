package handlers

import (
	"net/http"
	"strconv"

	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type EstimateHandler struct {
	repo      *repository.EstimateRepo
	orderRepo *repository.OrderRepo
}

func NewEstimateHandler(repo *repository.EstimateRepo, orderRepo *repository.OrderRepo) *EstimateHandler {
	return &EstimateHandler{repo: repo, orderRepo: orderRepo}
}

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

func (h *EstimateHandler) CatalogFlat(c *gin.Context) {
	items, err := h.repo.CatalogFlat(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []repository.ServiceCatalogItem{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

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

func (h *EstimateHandler) ColorList(c *gin.Context) {
	colors, err := h.repo.ColorList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if colors == nil {
		colors = []repository.ColorItem{}
	}
	c.JSON(http.StatusOK, gin.H{"data": colors})
}

func (h *EstimateHandler) EstimateGet(c *gin.Context) {
	services, materials, totalSvc, totalMat, err := h.repo.EstimateByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if services == nil {
		services = []repository.EstimateServiceRow{}
	}
	if materials == nil {
		materials = []repository.EstimateMaterialRow{}
	}
	c.JSON(http.StatusOK, gin.H{
		"services":        services,
		"materials":       materials,
		"total_services":  totalSvc,
		"total_materials": totalMat,
		"total":           totalSvc + totalMat,
	})
}

// cuttingSubGroups — подгруппы каталога которые относятся к Распилу.
// Все суммируются в один дочерний заказ с service_type="cutting".
var cuttingSubGroups = map[string]bool{
	"sawing":   true,
	"edging":   true,
	"drilling": true,
	"milling":  true,
	"gluing":   true,
	"packing":  true,
	"design":   true,
	"other":    true,
}

func (h *EstimateHandler) EstimateSave(c *gin.Context) {
	claims := middleware.GetClaims(c)
	orderID := c.Param("order_id")

	var req repository.SaveEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	groupAmounts, err := h.repo.SaveEstimate(c, orderID, claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.orderRepo != nil {
		parentOrder, err := h.orderRepo.OrderByID(c, orderID)
		if err == nil && parentOrder != nil && (parentOrder.OrderType == "workshop" || parentOrder.OrderType == "external") {

			// Суммируем все подгруппы Распила → один дочерний заказ "cutting"
			var cuttingTotal float64
			for group, amount := range groupAmounts {
				if cuttingSubGroups[group] {
					cuttingTotal += amount
				}
			}
			if cuttingTotal > 0 {
				_ = h.orderRepo.UpsertServiceLink(
					c, orderID, parentOrder,
					"cutting", "cutting",
					cuttingTotal, claims.UserID,
				)
			}

			// Остальные группы (ЧПУ, Покраска, Мягкая мебель и т.д.)
			for group, amount := range groupAmounts {
				if cuttingSubGroups[group] {
					continue
				}
				orderType, ok := models.EstimateGroupToOrderType[group]
				if !ok || amount <= 0 {
					continue
				}
				if err := h.orderRepo.UpsertServiceLink(
					c, orderID, parentOrder,
					group, orderType,
					amount, claims.UserID,
				); err != nil {
					_ = err
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}