package handlers

import (
	"net/http"
	"time"

	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

// ── Расходы цеха ─────────────────────────────────────────────

type WorkshopExpenseHandler struct {
	repo *repository.WorkshopExpenseRepo
}

func NewWorkshopExpenseHandler(repo *repository.WorkshopExpenseRepo) *WorkshopExpenseHandler {
	return &WorkshopExpenseHandler{repo: repo}
}

func (h *WorkshopExpenseHandler) List(c *gin.Context) {
	expenses, total, err := h.repo.List(c, c.Query("from"), c.Query("to"), c.Query("category"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if expenses == nil {
		expenses = []models.WorkshopExpense{}
	}
	c.JSON(http.StatusOK, gin.H{"data": expenses, "total": total})
}

func (h *WorkshopExpenseHandler) Create(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateWorkshopExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.Create(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *WorkshopExpenseHandler) Update(c *gin.Context) {
	var req models.UpdateWorkshopExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Update(c, c.Param("id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

func (h *WorkshopExpenseHandler) Delete(c *gin.Context) {
	if err := h.repo.Delete(c, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

func (h *WorkshopExpenseHandler) Categories(c *gin.Context) {
	cats, err := h.repo.Categories(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cats == nil {
		cats = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"data": cats})
}

// ── Назначение на этапы + Табель ─────────────────────────────

type StageAssigneeHandler struct {
	repo *repository.StageAssigneeRepo
}

func NewStageAssigneeHandler(repo *repository.StageAssigneeRepo) *StageAssigneeHandler {
	return &StageAssigneeHandler{repo: repo}
}

// ── Этапы проектов ────────────────────────────────────────────

func (h *StageAssigneeHandler) ProjectStageAssignees(c *gin.Context) {
	assignees, err := h.repo.ProjectStageAssignees(c, c.Param("stage_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if assignees == nil {
		assignees = []models.StageAssignee{}
	}
	c.JSON(http.StatusOK, gin.H{"data": assignees})
}

func (h *StageAssigneeHandler) ProjectStageSync(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.AssignStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.ProjectStageSync(c, c.Param("stage_id"), claims.UserID, req.UserIDs, req.AssemblyPercents); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

// ── Этапы заказов ─────────────────────────────────────────────

func (h *StageAssigneeHandler) OrderStageAssignees(c *gin.Context) {
	assignees, err := h.repo.OrderStageAssignees(c, c.Param("stage_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if assignees == nil {
		assignees = []models.StageAssignee{}
	}
	c.JSON(http.StatusOK, gin.H{"data": assignees})
}

func (h *StageAssigneeHandler) OrderStageSync(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.AssignStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.OrderStageSync(c, c.Param("stage_id"), claims.UserID, req.UserIDs, req.AssemblyPercents); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

// ── Табель ────────────────────────────────────────────────────

func (h *StageAssigneeHandler) TimesheetList(c *gin.Context) {
	entries, err := h.repo.TimesheetList(c, c.Query("from"), c.Query("to"), c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []models.Timesheet{}
	}
	c.JSON(http.StatusOK, gin.H{"data": entries})
}

func (h *StageAssigneeHandler) TimesheetSummary(c *gin.Context) {
	summary, err := h.repo.TimesheetSummary(c, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if summary == nil {
		summary = []models.TimesheetSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"data": summary})
}

func (h *StageAssigneeHandler) TimesheetCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateTimesheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.TimesheetCreate(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *StageAssigneeHandler) TimesheetDelete(c *gin.Context) {
	if err := h.repo.TimesheetDelete(c, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

func (h *StageAssigneeHandler) TimesheetAutoFill(c *gin.Context) {
	n, err := h.repo.AutoFillFromOrderStages(c, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"added": n})
}

// ── Выплаты зарплат ───────────────────────────────────────────

func (h *StageAssigneeHandler) SalaryPaymentList(c *gin.Context) {
	payments, err := h.repo.SalaryPaymentList(c, c.Query("from"), c.Query("to"), c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if payments == nil {
		payments = []models.SalaryPayment{}
	}
	c.JSON(http.StatusOK, gin.H{"data": payments})
}

func (h *StageAssigneeHandler) SalaryPaymentCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateSalaryPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверка лимита аванса (до 15-го — max 50% от зарплаты)
	if req.PaymentType == "advance" {
		today := time.Now().Day()
		if today <= 15 {
			salary, hourlyRate, paidAdvance, _ := h.repo.GetUserSalaryInfo(
				c, req.UserID, req.PeriodFrom, req.PeriodTo,
			)
			baseSalary := salary
			if baseSalary == 0 {
				baseSalary = hourlyRate * 176
			}
			maxAdvance := baseSalary * 0.5
			if paidAdvance+req.Amount > maxAdvance {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":        "Аванс не может превышать 50% зарплаты до 15-го числа",
					"max_advance":  maxAdvance,
					"paid_advance": paidAdvance,
					"available":    maxAdvance - paidAdvance,
				})
				return
			}
		}
	}

	id, err := h.repo.SalaryPaymentCreate(c, req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *StageAssigneeHandler) SalaryPaymentDelete(c *gin.Context) {
	if err := h.repo.SalaryPaymentDelete(c, c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}