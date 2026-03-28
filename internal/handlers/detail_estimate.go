	package handlers

	import (
		"net/http"

		"jevon/internal/middleware"
		"jevon/internal/repository"

		"github.com/gin-gonic/gin"
	)

	type DetailEstimateHandler struct {
		repo *repository.DetailEstimateRepo
	}

	func NewDetailEstimateHandler(repo *repository.DetailEstimateRepo) *DetailEstimateHandler {
		return &DetailEstimateHandler{repo: repo}
	}

	// GET /api/orders/:order_id/detail-estimate
	// Возвращает все разделы сметы заказа
	func (h *DetailEstimateHandler) GetEstimate(c *gin.Context) {
		sections, err := h.repo.GetByOrder(c, c.Param("order_id"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if sections == nil {
			sections = []repository.DetailEstimateSection{}
		}
		c.JSON(http.StatusOK, gin.H{
			"data":           sections,
			"service_labels": repository.ServiceTypeLabels,
			"subtitles":      repository.ServiceTypeSubtitles,
		})
	}

	// POST /api/orders/:order_id/detail-estimate
	func (h *DetailEstimateHandler) SaveSection(c *gin.Context) {
		claims := middleware.GetClaims(c)
		var req repository.SaveDetailEstimateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.repo.SaveSection(c, c.Param("order_id"), claims.UserID, req); err != nil {
			// Логируем полную ошибку
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":        err.Error(),
				"order_id":     c.Param("order_id"),
				"service_type": req.ServiceType,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "saved"})
	}

	// DELETE /api/orders/:order_id/detail-estimate/:service_type
	func (h *DetailEstimateHandler) DeleteSection(c *gin.Context) {
		if err := h.repo.DeleteSection(c, c.Param("order_id"), c.Param("service_type")); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}