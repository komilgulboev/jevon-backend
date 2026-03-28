package models

import "time"

// ── Клиенты ──────────────────────────────────────────────

type Client struct {
	ID        string    `json:"id"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	Phone2    string    `json:"phone2"`
	Company   string    `json:"company"`
	Address   string    `json:"address"`
	Notes     string    `json:"notes"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateClientRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Phone    string `json:"phone"     binding:"required"`
	Phone2   string `json:"phone2"`
	Company  string `json:"company"`
	Address  string `json:"address"`
	Notes    string `json:"notes"`
}

type UpdateClientRequest struct {
	FullName *string `json:"full_name"`
	Phone    *string `json:"phone"`
	Phone2   *string `json:"phone2"`
	Company  *string `json:"company"`
	Address  *string `json:"address"`
	Notes    *string `json:"notes"`
}

// ── Прайслист ─────────────────────────────────────────────

type PriceItem struct {
	ID          int     `json:"id"`
	OrderType   string  `json:"order_type"`
	ServiceType string  `json:"service_type"`
	Name        string  `json:"name"`
	Unit        string  `json:"unit"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	IsActive    bool    `json:"is_active"`
	Notes       string  `json:"notes"`
}

type UpdatePriceRequest struct {
	Price    *float64 `json:"price"`
	IsActive *bool    `json:"is_active"`
	Notes    *string  `json:"notes"`
}

// ── Заказ ─────────────────────────────────────────────────

type Order struct {
	ID            string    `json:"id"`
	OrderNumber   int       `json:"order_number"`
	OrderType     string    `json:"order_type"`
	ClientID      string    `json:"client_id"`
	ClientName    string    `json:"client_name"`
	ClientPhone   string    `json:"client_phone"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Address       string    `json:"address"`
	LocationURL   string    `json:"location_url"`
	CurrentStage  string    `json:"current_stage"`
	Status        string    `json:"status"`
	Priority      string    `json:"priority"`
	Deadline      *string   `json:"deadline"`
	StartedAt     *string   `json:"started_at"`
	FinishedAt    *string   `json:"finished_at"`
	EstimatedCost float64   `json:"estimated_cost"`
	FinalCost     float64   `json:"final_cost"`
	PaidAmount    float64   `json:"paid_amount"`
	PaymentStatus string    `json:"payment_status"`
	DriverID      string    `json:"driver_id"`
	Vehicle       string    `json:"vehicle"`
	DistanceKm    float64   `json:"distance_km"`
	FuelExpense   float64   `json:"fuel_expense"`
	ManagerID     string    `json:"manager_id"`
	ManagerName   string    `json:"manager_name"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	OrderType     string  `json:"order_type"  binding:"required"`
	ClientID      string  `json:"client_id"`
	ClientName    string  `json:"client_name"`
	ClientPhone   string  `json:"client_phone"`
	Title         string  `json:"title"       binding:"required"`
	Description   string  `json:"description"`
	Address       string  `json:"address"`
	LocationURL   string  `json:"location_url"`
	Priority      string  `json:"priority"`
	Deadline      *string `json:"deadline"`
	EstimatedCost float64 `json:"estimated_cost"`
	ManagerID     string  `json:"manager_id"`
}

type UpdateOrderRequest struct {
	Title         *string  `json:"title"`
	Description   *string  `json:"description"`
	Address       *string  `json:"address"`
	LocationURL   *string  `json:"location_url"`
	Status        *string  `json:"status"`
	Priority      *string  `json:"priority"`
	Deadline      *string  `json:"deadline"`
	EstimatedCost *float64 `json:"estimated_cost"`
	FinalCost     *float64 `json:"final_cost"`
	ManagerID     *string  `json:"manager_id"`
	DriverID      *string  `json:"driver_id"`
	Vehicle       *string  `json:"vehicle"`
	DistanceKm    *float64 `json:"distance_km"`
	FuelExpense   *float64 `json:"fuel_expense"`
}

// ── Этапы заказа ─────────────────────────────────────────

type OrderStage struct {
	ID           string     `json:"id"`
	OrderID      string     `json:"order_id"`
	Stage        string     `json:"stage"`
	StageOrder   int        `json:"stage_order"`
	Status       string     `json:"status"`
	AssignedTo   string     `json:"assigned_to"`
	AssigneeName string     `json:"assignee_name"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Notes        string     `json:"notes"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type UpdateOrderStageRequest struct {
	Status     *string `json:"status"`
	AssignedTo *string `json:"assigned_to"`
	Notes      *string `json:"notes"`
}

type CompleteOrderStageRequest struct {
	Notes   string `json:"notes"`
	Comment string `json:"comment"`
}

// ── Оплаты ───────────────────────────────────────────────

type OrderPayment struct {
	ID           string    `json:"id"`
	OrderID      string    `json:"order_id"`
	Amount       float64   `json:"amount"`
	PaymentType  string    `json:"payment_type"`
	PaidAt       time.Time `json:"paid_at"`
	Notes        string    `json:"notes"`
	ReceivedBy   string    `json:"received_by"`
	ReceiverName string    `json:"receiver_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreatePaymentRequest struct {
	Amount      float64 `json:"amount"       binding:"required"`
	PaymentType string  `json:"payment_type"`
	Notes       string  `json:"notes"`
}

// ── Расчёты ───────────────────────────────────────────────

type OrderCalculation struct {
	ID             string  `json:"id"`
	OrderID        string  `json:"order_id"`
	StageID        string  `json:"stage_id"`
	TotalAreaM2    float64 `json:"total_area_m2"`
	PaintKg        float64 `json:"paint_kg"`
	PrimerKg       float64 `json:"primer_kg"`
	PaintType      string  `json:"paint_type"`
	CNCType        string  `json:"cnc_type"`
	CNCTimeHours   float64 `json:"cnc_time_hours"`
	SheetCount     int     `json:"sheet_count"`
	CutLengthM     float64 `json:"cut_length_m"`
	ClientMaterial bool    `json:"client_material"`
	ClientMatDesc  string  `json:"client_material_desc"`
	CalculatedCost float64 `json:"calculated_cost"`
	CalculatedBy   string  `json:"calculated_by"`
	CalculatedAt   string  `json:"calculated_at"`
	Notes          string  `json:"notes"`
}

type CreateCalculationRequest struct {
	StageID        string  `json:"stage_id"`
	TotalAreaM2    float64 `json:"total_area_m2"`
	PaintType      string  `json:"paint_type"`
	CNCType        string  `json:"cnc_type"`
	CNCTimeHours   float64 `json:"cnc_time_hours"`
	SheetCount     int     `json:"sheet_count"`
	CutLengthM     float64 `json:"cut_length_m"`
	ClientMaterial bool    `json:"client_material"`
	ClientMatDesc  string  `json:"client_material_desc"`
	Notes          string  `json:"notes"`
}

// ── Позиции заказа ────────────────────────────────────────

type OrderItem struct {
	ID           string  `json:"id"`
	OrderID      string  `json:"order_id"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	Depth        float64 `json:"depth"`
	AreaM2       float64 `json:"area_m2"`
	MaterialName string  `json:"material_name"`
	UnitPrice    float64 `json:"unit_price"`
	TotalPrice   float64 `json:"total_price"`
	Notes        string  `json:"notes"`
}

type CreateOrderItemRequest struct {
	Name         string  `json:"name"     binding:"required"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	Depth        float64 `json:"depth"`
	MaterialName string  `json:"material_name"`
	UnitPrice    float64 `json:"unit_price"`
	Notes        string  `json:"notes"`
}

// ── Комментарии ───────────────────────────────────────────

type OrderComment struct {
	ID         string    `json:"id"`
	OrderID    string    `json:"order_id"`
	StageID    string    `json:"stage_id"`
	Text       string    `json:"text"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateCommentRequest struct {
	StageID string `json:"stage_id"`
	Text    string `json:"text" binding:"required"`
}

// ── История ───────────────────────────────────────────────

type OrderHistory struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	FromStage   string    `json:"from_stage"`
	ToStage     string    `json:"to_stage"`
	ChangedBy   string    `json:"changed_by"`
	ChangerName string    `json:"changer_name"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Лейблы этапов ────────────────────────────────────────

var OrderTypeLabels = map[string]string{
	"workshop":       "Заказ цеха",
	"cutting":        "Распил",
	"painting":       "Покраска",
	"cnc":            "ЧПУ",
	"soft_fabric":    "Мягкая мебель (обивка)",
	"soft_furniture": "Производство мебели",
}

var StageLabelsByType = map[string]map[string]string{
	"workshop": {
		"intake": "Приём заказа", "measure": "Замер",
		"design": "Дизайн/Смета", "purchase": "Закупка",
		"production": "Производство", "assembly": "Сборка",
		"delivery": "Доставка", "handover": "Сдача клиенту",
	},
	"cutting": {
		"intake": "Приём заказа", "material": "Приём материала",
		"sawing": "Распил", "edging": "Кромкование",
		"drilling": "Присадка", "packing": "Упаковка",
		"shipment": "Отгрузка",
	},
	"painting": {
		"intake": "Приём деталей", "calculate": "Расчёт",
		"sanding": "Шлифовка", "priming": "Грунтовка",
		"painting": "Покраска", "delivery": "Выдача",
	},
	"cnc": {
		"intake": "Приём заказа", "calculate": "Расчёт",
		"cnc_work": "Фрезеровка", "delivery": "Выдача",
	},
	"soft_fabric": {
		"intake": "Приём заказа", "calculate": "Расчёт",
		"assign": "Назначение мастера", "work": "Работа",
		"delivery": "Выдача",
	},
	"soft_furniture": {
		"intake": "Приём заказа", "design": "Дизайн",
		"purchase": "Закупка", "production": "Производство",
		"delivery": "Доставка",
	},
}

var StageRoles = map[string][]string{
	"intake":      {"manager", "admin", "supervisor"},
	"measure":     {"manager", "admin", "supervisor"},
	"design":      {"designer", "admin", "supervisor"},
	"purchase":    {"manager", "admin", "supervisor"},
	"production":  {"master", "cutter", "admin", "supervisor"},
	"assembly":    {"assembler", "master", "admin", "supervisor"},
	"delivery":    {"driver", "admin", "supervisor"},
	"handover":    {"manager", "admin", "supervisor"},
	"material":    {"warehouse", "admin", "supervisor"},
	"sawing":      {"cutter", "master", "admin", "supervisor"},
	"edging":      {"cutter", "master", "admin", "supervisor"},
	"drilling":    {"cutter", "master", "admin", "supervisor"},
	"packing":     {"warehouse", "cutter", "admin", "supervisor"},
	"shipment":    {"manager", "warehouse", "admin", "supervisor"},
	"calculate":   {"manager", "master", "admin", "supervisor"},
	"sanding":     {"painter", "master", "admin", "supervisor"},
	"priming":     {"painter", "master", "admin", "supervisor"},
	"painting":    {"painter", "master", "admin", "supervisor"},
	"cnc_work":    {"cnc_operator", "admin", "supervisor"},
	"assign":      {"supervisor", "admin"},
	"work":        {"upholsterer", "master", "admin", "supervisor"},
}