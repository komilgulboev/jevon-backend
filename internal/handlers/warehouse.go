package handlers

import (
	"net/http"

	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type WarehouseHandler struct {
	repo *repository.WarehouseRepo
}

func NewWarehouseHandler(repo *repository.WarehouseRepo) *WarehouseHandler {
	return &WarehouseHandler{repo: repo}
}

// ─── Единицы измерения ────────────────────────────────────────

func (h *WarehouseHandler) UnitList(c *gin.Context) {
	units, err := h.repo.UnitList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if units == nil {
		units = []models.Unit{}
	}
	c.JSON(http.StatusOK, units)
}

// ─── Категории ────────────────────────────────────────────────

func (h *WarehouseHandler) CategoryList(c *gin.Context) {
	cats, err := h.repo.CategoryList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cats == nil {
		cats = []string{}
	}
	c.JSON(http.StatusOK, cats)
}

// ─── Номенклатура ─────────────────────────────────────────────

func (h *WarehouseHandler) ItemList(c *gin.Context) {
	items, err := h.repo.ItemList(c, c.Query("category"), c.Query("search"), c.Query("active"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []models.WarehouseItem{}
	}
	cats, _ := h.repo.CategoryList(c)
	if cats == nil {
		cats = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "categories": cats})
}

func (h *WarehouseHandler) ItemGet(c *gin.Context) {
	item, err := h.repo.ItemByID(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *WarehouseHandler) ItemCreate(c *gin.Context) {
	var req models.CreateWarehouseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.ItemCreate(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *WarehouseHandler) ItemUpdate(c *gin.Context) {
	var req models.UpdateWarehouseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.ItemUpdate(c, c.Param("id"), req); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

func (h *WarehouseHandler) ItemDelete(c *gin.Context) {
	result, err := h.repo.ItemDelete(c, c.Param("id"))
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result == "deactivated" {
		c.JSON(http.StatusOK, gin.H{"message": "деактивировано (есть движения по товару)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

// ─── Поставщики ───────────────────────────────────────────────

func (h *WarehouseHandler) SupplierList(c *gin.Context) {
	suppliers, err := h.repo.SupplierList(c, c.Query("search"), c.Query("active"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if suppliers == nil {
		suppliers = []models.Supplier{}
	}
	c.JSON(http.StatusOK, gin.H{"data": suppliers})
}

func (h *WarehouseHandler) SupplierGet(c *gin.Context) {
	s, err := h.repo.SupplierByID(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *WarehouseHandler) SupplierCreate(c *gin.Context) {
	var req models.CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.SupplierCreate(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *WarehouseHandler) SupplierUpdate(c *gin.Context) {
	var req models.UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SupplierUpdate(c, c.Param("id"), req); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

func (h *WarehouseHandler) SupplierDelete(c *gin.Context) {
	result, err := h.repo.SupplierDelete(c, c.Param("id"))
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result == "deactivated" {
		c.JSON(http.StatusOK, gin.H{"message": "деактивировано (есть накладные от поставщика)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

// ─── Платежи поставщику (общий расчёт) ───────────────────────

// GET /api/warehouse/suppliers/:id/payments
func (h *WarehouseHandler) SupplierPaymentHistory(c *gin.Context) {
	payments, err := h.repo.SupplierPaymentHistory(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payments})
}

// POST /api/warehouse/suppliers/:id/payments
func (h *WarehouseHandler) SupplierPaymentCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateSupplierPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.repo.SupplierPaymentCreate(c, c.Param("id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// DELETE /api/warehouse/suppliers/:id/payments/:payment_id
func (h *WarehouseHandler) SupplierPaymentDelete(c *gin.Context) {
	if err := h.repo.SupplierPaymentDelete(c, c.Param("payment_id")); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

// ─── Приходные накладные ──────────────────────────────────────

func (h *WarehouseHandler) ReceiptList(c *gin.Context) {
	receipts, err := h.repo.ReceiptList(c, c.Query("supplier_id"), c.Query("search"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if receipts == nil {
		receipts = []models.Receipt{}
	}
	c.JSON(http.StatusOK, gin.H{"data": receipts})
}

func (h *WarehouseHandler) ReceiptGet(c *gin.Context) {
	rec, err := h.repo.ReceiptByID(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *WarehouseHandler) ReceiptCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.ReceiptCreate(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *WarehouseHandler) ReceiptUpdate(c *gin.Context) {
	var req models.UpdateReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.ReceiptUpdate(c, c.Param("id"), req); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

func (h *WarehouseHandler) ReceiptDelete(c *gin.Context) {
	if err := h.repo.ReceiptDelete(c, c.Param("id")); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

func (h *WarehouseHandler) ReceiptItemAdd(c *gin.Context) {
	var req models.AddReceiptItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	itemID, err := h.repo.ReceiptItemAdd(c, c.Param("id"), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": itemID})
}

func (h *WarehouseHandler) ReceiptItemDelete(c *gin.Context) {
	if err := h.repo.ReceiptItemDelete(c, c.Param("id"), c.Param("item_id")); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

// ─── Платежи по конкретной накладной ─────────────────────────

func (h *WarehouseHandler) PaymentList(c *gin.Context) {
	payments, err := h.repo.ReceiptPaymentList(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if payments == nil {
		payments = []models.ReceiptPayment{}
	}
	c.JSON(http.StatusOK, gin.H{"data": payments})
}

func (h *WarehouseHandler) PaymentCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateReceiptPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.ReceiptPaymentCreate(c, c.Param("id"), claims.UserID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *WarehouseHandler) PaymentDelete(c *gin.Context) {
	if err := h.repo.ReceiptPaymentDelete(c, c.Param("payment_id")); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}