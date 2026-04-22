// internal/handlers/estimate_section_stages.go
package handlers

import (
	"net/http"

	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type EstimateSectionStagesHandler struct {
	repo *repository.EstimateSectionStagesRepo
}

func NewEstimateSectionStagesHandler(repo *repository.EstimateSectionStagesRepo) *EstimateSectionStagesHandler {
	return &EstimateSectionStagesHandler{repo: repo}
}

// GET /api/orders/:order_id/estimate-stages
func (h *EstimateSectionStagesHandler) GetByOrder(c *gin.Context) {
	stages, err := h.repo.GetByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if stages == nil {
		stages = []repository.EstimateSectionStage{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   stages,
		"labels": repository.SectionStageLabels,
	})
}

// PATCH /api/orders/:order_id/estimate-stages/:stage_id
func (h *EstimateSectionStagesHandler) UpdateStage(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.UpdateStage(c, c.Param("stage_id"), req.Status, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/orders/:order_id/estimate-stages/:stage_id/complete
func (h *EstimateSectionStagesHandler) CompleteStage(c *gin.Context) {
	if err := h.repo.CompleteStage(c, c.Param("stage_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
