package models

import "time"

// ─── Расходы цеха ─────────────────────────────────────────────

type WorkshopExpense struct {
	ID             string    `json:"id"`
	ExpenseDate    string    `json:"expense_date"`
	Category       string    `json:"category"`
	Description    string    `json:"description"`
	Amount         float64   `json:"amount"`
	Method         string    `json:"method"`
	CreatedBy      string    `json:"created_by"`
	CreatorName    string    `json:"creator_name"`
	OrderID        string    `json:"order_id"`
	OrderNumber    int       `json:"order_number"`
	OrderTitle     string    `json:"order_title"`
	ProjectID      string    `json:"project_id"`
	ProjectTitle   string    `json:"project_title"`
	LinkedUserID   string    `json:"linked_user_id"`
	LinkedUserName string    `json:"linked_user_name"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateWorkshopExpenseRequest struct {
	ExpenseDate    string  `json:"expense_date"`
	Category       string  `json:"category"    binding:"required"`
	Description    string  `json:"description"`
	Amount         float64 `json:"amount"      binding:"required"`
	Method         string  `json:"method"`
	OrderID        string  `json:"order_id"`
	ProjectID      string  `json:"project_id"`
	LinkedUserID   string  `json:"linked_user_id"`
}

type UpdateWorkshopExpenseRequest struct {
	ExpenseDate  *string  `json:"expense_date"`
	Category     *string  `json:"category"`
	Description  *string  `json:"description"`
	Amount       *float64 `json:"amount"`
	Method       *string  `json:"method"`
}

// ─── Мультиназначение этапов ──────────────────────────────────

type StageAssignee struct {
	ID               string    `json:"id"`
	StageID          string    `json:"stage_id"`
	UserID           string    `json:"user_id"`
	FullName         string    `json:"full_name"`
	RoleName         string    `json:"role_name"`
	AvatarURL        string    `json:"avatar_url"`
	AssemblyPercent  float64   `json:"assembly_percent"`
	AssignedAt       time.Time `json:"assigned_at"`
}

type AssignStageRequest struct {
	UserIDs         []string  `json:"user_ids"          binding:"required"`
	AssemblyPercents []float64 `json:"assembly_percents"` // параллельный массив процентов
}

// ─── Табель ───────────────────────────────────────────────────

type Timesheet struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	FullName   string    `json:"full_name"`
	RoleName   string    `json:"role_name"`
	WorkDate   string    `json:"work_date"`
	Hours      float64   `json:"hours"`
	CheckIn    string    `json:"check_in"`
	CheckOut   string    `json:"check_out"`
	SourceType string    `json:"source_type"`
	SourceID   string    `json:"source_id"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

type TimesheetSummary struct {
	UserID          string   `json:"user_id"`
	FullName        string   `json:"full_name"`
	RoleName        string   `json:"role_name"`
	TotalHours      float64  `json:"total_hours"`
	WorkDays        int      `json:"work_days"`
	Salary          *float64 `json:"salary"`
	HourlyRate      *float64 `json:"hourly_rate"`
	Calculated      float64  `json:"calculated"`
	AssemblyBonus   float64  `json:"assembly_bonus"`   // бонус от % сборки
	TotalToPay      float64  `json:"total_to_pay"`
	PaidAdvance     float64  `json:"paid_advance"`     // выплачено авансом
	PaidSalary      float64  `json:"paid_salary"`      // выплачено зарплатой
	TotalPaid       float64  `json:"total_paid"`
	Remaining       float64  `json:"remaining"`
}

type CreateTimesheetRequest struct {
	UserID     string  `json:"user_id"    binding:"required"`
	WorkDate   string  `json:"work_date"  binding:"required"`
	Hours      float64 `json:"hours"`
	CheckIn    string  `json:"check_in"`
	CheckOut   string  `json:"check_out"`
	SourceType string  `json:"source_type"`
	SourceID   string  `json:"source_id"`
	Notes      string  `json:"notes"`
}

// ─── Выплаты зарплат ──────────────────────────────────────────

type SalaryPayment struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	FullName    string    `json:"full_name"`
	Amount      float64   `json:"amount"`
	PeriodFrom  string    `json:"period_from"`
	PeriodTo    string    `json:"period_to"`
	PaymentType string    `json:"payment_type"` // salary | advance
	Method      string    `json:"method"`
	Notes       string    `json:"notes"`
	PaidBy      string    `json:"paid_by"`
	PaidByName  string    `json:"paid_by_name"`
	PaidAt      time.Time `json:"paid_at"`
}

type CreateSalaryPaymentRequest struct {
	UserID      string  `json:"user_id"      binding:"required"`
	Amount      float64 `json:"amount"       binding:"required"`
	PeriodFrom  string  `json:"period_from"  binding:"required"`
	PeriodTo    string  `json:"period_to"    binding:"required"`
	PaymentType string  `json:"payment_type"` // salary | advance
	Method      string  `json:"method"`
	Notes       string  `json:"notes"`
}