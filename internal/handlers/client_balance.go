package handlers

import (
	"net/http"

	"jevon/internal/middleware"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type ClientBalanceHandler struct {
	repo *repository.ClientBalanceRepo
}

func NewClientBalanceHandler(repo *repository.ClientBalanceRepo) *ClientBalanceHandler {
	return &ClientBalanceHandler{repo: repo}
}

// GET /api/clients/debt?search=&filter=debt|credit|clear
func (h *ClientBalanceHandler) DebtList(c *gin.Context) {
	clients, err := h.repo.ClientDebtList(c, c.Query("search"), c.Query("filter"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": clients})
}

// GET /api/clients/:id/orders
func (h *ClientBalanceHandler) ClientOrders(c *gin.Context) {
	orders, err := h.repo.ClientOrders(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

// GET /api/clients/:id/payments
func (h *ClientBalanceHandler) PaymentHistory(c *gin.Context) {
	payments, err := h.repo.ClientPaymentHistory(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payments})
}

// POST /api/clients/:id/payments
func (h *ClientBalanceHandler) PaymentCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req repository.CreateClientPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.repo.ClientPaymentCreate(c, c.Param("id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// DELETE /api/clients/:id/payments/:payment_id
func (h *ClientBalanceHandler) PaymentDelete(c *gin.Context) {
	if err := h.repo.ClientPaymentDelete(c, c.Param("payment_id")); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}