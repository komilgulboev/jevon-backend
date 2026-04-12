package handlers

import (
	"net/http"

	"jevon/internal/middleware"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type OutgoingInvoiceHandler struct {
	repo *repository.OutgoingInvoiceRepo
}

func NewOutgoingInvoiceHandler(repo *repository.OutgoingInvoiceRepo) *OutgoingInvoiceHandler {
	return &OutgoingInvoiceHandler{repo: repo}
}

func (h *OutgoingInvoiceHandler) List(c *gin.Context) {
	invoices, err := h.repo.List(c, c.Query("status"), c.Query("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if invoices == nil {
		invoices = []repository.OutgoingInvoice{}
	}
	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

func (h *OutgoingInvoiceHandler) Get(c *gin.Context) {
	inv, err := h.repo.GetByID(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if inv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *OutgoingInvoiceHandler) Create(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req repository.CreateOutgoingInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Добавьте хотя бы один товар"})
		return
	}
	inv, err := h.repo.Create(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, inv)
}

func (h *OutgoingInvoiceHandler) Confirm(c *gin.Context) {
	result, err := h.repo.Confirm(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Если pending_purchase — возвращаем 200 с деталями дефицита
	// Фронтенд сам решает как показать сообщение
	c.JSON(http.StatusOK, result)
}

func (h *OutgoingInvoiceHandler) Cancel(c *gin.Context) {
	if err := h.repo.Cancel(c, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "отменено"})
}