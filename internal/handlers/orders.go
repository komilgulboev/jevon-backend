package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	repo *repository.OrderRepo
}

func NewOrderHandler(repo *repository.OrderRepo) *OrderHandler {
	return &OrderHandler{repo: repo}
}

// GET /api/clients
func (h *OrderHandler) ClientList(c *gin.Context) {
	clients, err := h.repo.ClientList(c, c.Query("search"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if clients == nil { clients = []models.Client{} }
	c.JSON(http.StatusOK, gin.H{"data": clients})
}

// POST /api/clients
func (h *OrderHandler) ClientCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.ClientCreate(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// PATCH /api/clients/:id
func (h *OrderHandler) ClientUpdate(c *gin.Context) {
	var req models.UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.ClientUpdate(c, c.Param("id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// GET /api/price-list
func (h *OrderHandler) PriceList(c *gin.Context) {
	items, err := h.repo.PriceList(c, c.Query("order_type"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil { items = []models.PriceItem{} }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// PATCH /api/price-list/:id
func (h *OrderHandler) PriceUpdate(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.UpdatePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.PriceUpdate(c, id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// GET /api/orders
func (h *OrderHandler) OrderList(c *gin.Context) {
	claims := middleware.GetClaims(c)
	orders, err := h.repo.OrderList(c,
		claims.UserID, claims.RoleName,
		c.Query("order_type"),
		c.Query("status"),
		c.Query("payment_status"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if orders == nil { orders = []models.Order{} }
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

// GET /api/orders/:order_id
func (h *OrderHandler) OrderGet(c *gin.Context) {
	order, err := h.repo.OrderByID(c, c.Param("order_id"))
	if err != nil || order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

// POST /api/orders
func (h *OrderHandler) OrderCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.OrderCreate(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// PATCH /api/orders/:order_id
func (h *OrderHandler) OrderUpdate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.OrderUpdate(c, c.Param("order_id"), req, claims.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DELETE /api/orders/:order_id
func (h *OrderHandler) OrderCancel(c *gin.Context) {
	h.repo.OrderCancel(c, c.Param("order_id"))
	c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
}

// GET /api/orders/stats
func (h *OrderHandler) OrderStats(c *gin.Context) {
	stats, err := h.repo.Stats(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GET /api/orders/:order_id/stages
func (h *OrderHandler) StagesList(c *gin.Context) {
	stages, err := h.repo.StagesByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if stages == nil { stages = []models.OrderStage{} }
	c.JSON(http.StatusOK, gin.H{"data": stages})
}

// PATCH /api/orders/:order_id/stages/:stage_id
func (h *OrderHandler) StageUpdate(c *gin.Context) {
	var req models.UpdateOrderStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.StageUpdate(c, c.Param("stage_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// POST /api/orders/:order_id/stages/:stage_id/complete
func (h *OrderHandler) StageComplete(c *gin.Context) {
	var req models.CompleteOrderStageRequest
	c.ShouldBindJSON(&req)
	if err := h.repo.StageComplete(c, c.Param("stage_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stage completed"})
}

// GET /api/orders/:order_id/payments
func (h *OrderHandler) PaymentsList(c *gin.Context) {
	payments, err := h.repo.PaymentsByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if payments == nil { payments = []models.OrderPayment{} }
	c.JSON(http.StatusOK, gin.H{"data": payments})
}

// POST /api/orders/:order_id/payments
func (h *OrderHandler) PaymentCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.PaymentCreate(c, c.Param("order_id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GET /api/orders/:order_id/calculation
func (h *OrderHandler) CalculationGet(c *gin.Context) {
	calc, err := h.repo.CalculationByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if calc == nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, calc)
}

// POST /api/orders/:order_id/calculation
func (h *OrderHandler) CalculationCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.CalculationCreate(c, c.Param("order_id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GET /api/orders/:order_id/comments
func (h *OrderHandler) CommentsList(c *gin.Context) {
	comments, err := h.repo.CommentsByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if comments == nil { comments = []models.OrderComment{} }
	c.JSON(http.StatusOK, gin.H{"data": comments})
}

// POST /api/orders/:order_id/comments
func (h *OrderHandler) CommentCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.CommentCreate(c, c.Param("order_id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GET /api/orders/:order_id/history
func (h *OrderHandler) History(c *gin.Context) {
	history, err := h.repo.History(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if history == nil { history = []models.OrderHistory{} }
	c.JSON(http.StatusOK, gin.H{"data": history})
}

// GET /api/orders/:order_id/materials
func (h *OrderHandler) MaterialsList(c *gin.Context) {
	materials, total, err := h.repo.MaterialsByOrder(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if materials == nil { materials = []repository.OrderMaterial{} }
	c.JSON(http.StatusOK, gin.H{
		"data":        materials,
		"total_price": total,
	})
}

// POST /api/orders/:order_id/materials
func (h *OrderHandler) MaterialCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req repository.CreateOrderMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.MaterialCreate(c, c.Param("order_id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// DELETE /api/orders/:order_id/materials/:material_id
func (h *OrderHandler) MaterialDelete(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if err := h.repo.MaterialDelete(c, c.Param("material_id"), claims.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GET /api/materials/catalog
func (h *OrderHandler) MaterialsCatalog(c *gin.Context) {
	items, err := h.repo.MaterialsCatalog(c, c.Query("search"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil { items = []repository.CatalogMaterial{} }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// GET /api/orders/labels
func (h *OrderHandler) Labels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"order_types":  models.OrderTypeLabels,
		"stage_labels": models.StageLabelsByType,
		"stage_roles":  models.StageRoles,
	})
}

// GET /api/orders/:order_id/expenses
func (h *OrderHandler) ExpensesList(c *gin.Context) {
	expenses, err := h.repo.ExpensesList(c, c.Param("order_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if expenses == nil {
		expenses = []repository.Expense{}
	}

	// Считаем итог расходов
	total := 0.0
	for _, e := range expenses {
		total += e.Amount
	}

	c.JSON(http.StatusOK, gin.H{"data": expenses, "total": total})
}

// POST /api/orders/:order_id/expenses
func (h *OrderHandler) ExpenseCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req repository.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.ExpenseCreate(c, c.Param("order_id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.repo.LogHistory(c, c.Param("order_id"), "expense", "expense", claims.UserID,
		fmt.Sprintf("💸 Расход: %s — %.0f сом.", req.Name, req.Amount))
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// DELETE /api/orders/:order_id/expenses/:expense_id
func (h *OrderHandler) ExpenseDelete(c *gin.Context) {
	if err := h.repo.ExpenseDelete(c, c.Param("expense_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}