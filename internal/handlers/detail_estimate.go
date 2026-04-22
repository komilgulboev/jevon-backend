package handlers

import (
	"log"
	"net/http"

	"jevon/internal/middleware"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type DetailEstimateHandler struct {
	repo      *repository.DetailEstimateRepo
	orderRepo *repository.OrderRepo
}

func NewDetailEstimateHandler(repo *repository.DetailEstimateRepo, orderRepo *repository.OrderRepo) *DetailEstimateHandler {
	return &DetailEstimateHandler{repo: repo, orderRepo: orderRepo}
}

// GET /api/orders/:order_id/detail-estimate
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
	orderID := c.Param("order_id")

	var req repository.SaveDetailEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[DetailEstimate] START: order=%s service=%s rows=%d", orderID, req.ServiceType, len(req.Rows))

	// 1. Сохраняем смету
	totalPrice, err := h.repo.SaveSection(c, orderID, claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":        err.Error(),
			"order_id":     orderID,
			"service_type": req.ServiceType,
		})
		return
	}

	log.Printf("[DetailEstimate] SaveSection OK: total=%.2f", totalPrice)

	// Если нет orderRepo — выходим
	if h.orderRepo == nil {
		c.JSON(http.StatusOK, gin.H{"message": "saved"})
		return
	}

	// 2. Если сумма 0 — не создаём/не обновляем дочерний заказ
	if totalPrice <= 0 {
		log.Printf("[DetailEstimate] totalPrice=0, skip UpsertServiceLink")
		c.JSON(http.StatusOK, gin.H{"message": "saved"})
		return
	}

	// 3. Определяем тип дочернего заказа
	orderType, ok := repository.DetailServiceToOrderType[req.ServiceType]
	if !ok {
		log.Printf("[DetailEstimate] WARNING: no mapping for service_type=%s", req.ServiceType)
		c.JSON(http.StatusOK, gin.H{"message": "saved"})
		return
	}

	// 4. Получаем родительский заказ
	parentOrder, err := h.orderRepo.OrderByID(c, orderID)
	if err != nil || parentOrder == nil {
		log.Printf("[DetailEstimate] OrderByID error: %v", err)
		c.JSON(http.StatusOK, gin.H{"message": "saved"})
		return
	}

	// Создаём дочерние заказы только для workshop и external
	if parentOrder.OrderType != "workshop" && parentOrder.OrderType != "external" {
		log.Printf("[DetailEstimate] skip: not workshop/external, type=%s", parentOrder.OrderType)
		c.JSON(http.StatusOK, gin.H{"message": "saved"})
		return
	}

	// 5. Создаём/обновляем связанный дочерний заказ
	log.Printf("[DetailEstimate] UpsertServiceLink: parent=%s service=%s type=%s amount=%.2f",
		orderID, req.ServiceType, orderType, totalPrice)

	if err := h.orderRepo.UpsertServiceLink(
		c, orderID, parentOrder,
		req.ServiceType, orderType,
		totalPrice, claims.UserID,
	); err != nil {
		log.Printf("[DetailEstimate] UpsertServiceLink ERROR: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[DetailEstimate] UpsertServiceLink OK")

	// 6. Копируем строки в дочерний заказ
	links, err := h.orderRepo.ServiceLinksByParent(c, orderID)
	if err == nil {
		for _, link := range links {
			if link.ServiceType == req.ServiceType {
				log.Printf("[DetailEstimate] CopyToChildOrder → %s", link.ChildOrderID)
				if err := h.repo.CopyToChildOrder(c, orderID, link.ChildOrderID, req.ServiceType); err != nil {
					log.Printf("[DetailEstimate] Copy ERROR: %v", err)
				}
				break
			}
		}
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