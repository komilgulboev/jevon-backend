package models

// ─── Единицы измерения ────────────────────────────────────────

type Unit struct {
	ID   int    `db:"id"   json:"id"`
	Name string `db:"name" json:"name"`
}

// ─── Поставщики ───────────────────────────────────────────────

type Supplier struct {
	ID        string  `db:"id"          json:"id"`
	Name      string  `db:"name"        json:"name"`
	Phone     string  `db:"phone"       json:"phone"`
	Email     string  `db:"email"       json:"email"`
	Address   string  `db:"address"     json:"address"`
	WhatsApp  string  `db:"whatsapp"    json:"whatsapp"`
	Telegram  string  `db:"telegram"    json:"telegram"`
	Notes     string  `db:"notes"       json:"notes"`
	IsActive  bool    `db:"is_active"   json:"is_active"`
	CreatedAt string  `db:"created_at"  json:"created_at"`
	// Из supplier_debt VIEW
	TotalDebt    float64 `db:"total_debt"    json:"total_debt"`
	TotalAmount  float64 `db:"total_amount"  json:"total_amount"`
	TotalPaid    float64 `db:"total_paid"    json:"total_paid"`
	UnpaidCount  int     `db:"unpaid_count"  json:"unpaid_count"`
}

type CreateSupplierRequest struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Address  string `json:"address"`
	WhatsApp string `json:"whatsapp"`
	Telegram string `json:"telegram"`
	Notes    string `json:"notes"`
}

type UpdateSupplierRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Address  string `json:"address"`
	WhatsApp string `json:"whatsapp"`
	Telegram string `json:"telegram"`
	Notes    string `json:"notes"`
	IsActive *bool  `json:"is_active"`
}

// ─── Платёж поставщику (общий) ───────────────────────────────

type SupplierPayment struct {
	ID                string   `db:"id"                 json:"id"`
	SupplierID        string   `db:"supplier_id"        json:"supplier_id"`
	ReceiptID         *string  `db:"receipt_id"         json:"receipt_id"`
	Amount            float64  `db:"amount"             json:"amount"`
	PaymentMethod     string   `db:"payment_method"     json:"payment_method"`
	PaidAt            string   `db:"paid_at"            json:"paid_at"`
	Notes             string   `db:"notes"              json:"notes"`
	IsSupplierPayment bool     `db:"is_supplier_payment" json:"is_supplier_payment"`
	CreatedBy         *string  `db:"created_by"         json:"created_by"`
	CreatedAt         string   `db:"created_at"         json:"created_at"`
	// Для истории — имя накладной
	ReceiptNumber     string   `db:"receipt_number"     json:"receipt_number"`
}

type CreateSupplierPaymentRequest struct {
	Amount        float64 `json:"amount"         binding:"required,gt=0"`
	PaymentMethod string  `json:"payment_method" binding:"required"`
	PaidAt        string  `json:"paid_at"`
	Notes         string  `json:"notes"`
}

// Результат распределения платежа
type PaymentDistribution struct {
	ReceiptID     string  `json:"receipt_id"`
	ReceiptNumber string  `json:"receipt_number"`
	Applied       float64 `json:"applied"`
	Remaining     float64 `json:"remaining"`
	Status        string  `json:"status"` // paid / partial
}

type SupplierPaymentResult struct {
	TotalPaid      float64               `json:"total_paid"`
	TotalDebtBefore float64              `json:"total_debt_before"`
	TotalDebtAfter float64               `json:"total_debt_after"`
	Distribution   []PaymentDistribution `json:"distribution"`
}

// ─── Долг поставщика ──────────────────────────────────────────

type SupplierDebt struct {
	SupplierID   string  `db:"supplier_id"   json:"supplier_id"`
	SupplierName string  `db:"supplier_name" json:"supplier_name"`
	TotalReceipts int    `db:"total_receipts" json:"total_receipts"`
	TotalAmount  float64 `db:"total_amount"  json:"total_amount"`
	TotalPaid    float64 `db:"total_paid"    json:"total_paid"`
	TotalDebt    float64 `db:"total_debt"    json:"total_debt"`
	UnpaidCount  int     `db:"unpaid_count"  json:"unpaid_count"`
}

// ─── Номенклатура ─────────────────────────────────────────────

type WarehouseItem struct {
	ID       string  `db:"id"        json:"id"`
	Name     string  `db:"name"      json:"name"`
	Article  string  `db:"article"   json:"article"`
	Category string  `db:"category"  json:"category"`
	UnitID   *int    `db:"unit_id"   json:"unit_id"`
	UnitName string  `db:"unit"      json:"unit"`
	MinStock float64 `db:"min_stock" json:"min_stock"`
	Notes    string  `db:"notes"     json:"notes"`
	IsActive bool    `db:"is_active" json:"is_active"`
	TotalIn  float64 `db:"total_in"  json:"total_in"`
	TotalOut float64 `db:"total_out" json:"total_out"`
	Balance  float64 `db:"balance"   json:"balance"`
	AvgPrice float64 `db:"avg_price" json:"avg_price"`
}

type CreateWarehouseItemRequest struct {
	Name     string  `json:"name" binding:"required"`
	Article  string  `json:"article"`
	Category string  `json:"category"`
	UnitID   *int    `json:"unit_id"`
	MinStock float64 `json:"min_stock"`
	Notes    string  `json:"notes"`
	IsActive *bool   `json:"is_active"`
}

type UpdateWarehouseItemRequest struct {
	Name     string  `json:"name"`
	Article  string  `json:"article"`
	Category string  `json:"category"`
	UnitID   *int    `json:"unit_id"`
	MinStock float64 `json:"min_stock"`
	Notes    string  `json:"notes"`
	IsActive *bool   `json:"is_active"`
}

// ─── Приходные накладные ──────────────────────────────────────

type ReceiptItem struct {
	ID        string  `db:"id"         json:"id"`
	ReceiptID string  `db:"receipt_id" json:"receipt_id"`
	ItemID    string  `db:"item_id"    json:"item_id"`
	ItemName  string  `db:"item_name"  json:"item_name"`
	Unit      string  `db:"unit"       json:"unit"`
	Quantity  float64 `db:"quantity"   json:"quantity"`
	Price     float64 `db:"price"      json:"price"`
	Total     float64 `db:"total"      json:"total"`
	Notes     string  `db:"notes"      json:"notes"`
}

type Receipt struct {
	ID            string           `db:"id"             json:"id"`
	Number        string           `db:"number"         json:"number"`
	SupplierID    *string          `db:"supplier_id"    json:"supplier_id"`
	SupplierName  string           `db:"supplier_name"  json:"supplier_name"`
	ReceiptDate   string           `db:"receipt_date"   json:"receipt_date"`
	TotalAmount   float64          `db:"total_amount"   json:"total_amount"`
	PaidAmount    float64          `db:"paid_amount"    json:"paid_amount"`
	PaymentStatus string           `db:"payment_status" json:"payment_status"`
	Notes         string           `db:"notes"          json:"notes"`
	CreatedBy     *string          `db:"created_by"     json:"created_by"`
	CreatedAt     string           `db:"created_at"     json:"created_at"`
	Items         []ReceiptItem    `db:"-"              json:"items"`
	Payments      []ReceiptPayment `db:"-"              json:"payments"`
}

type CreateReceiptItemInput struct {
	ItemID   string  `json:"item_id"  binding:"required"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	Price    float64 `json:"price"    binding:"gte=0"`
	Notes    string  `json:"notes"`
}

type CreateReceiptRequest struct {
	Number      string                   `json:"number"`
	SupplierID  *string                  `json:"supplier_id"`
	ReceiptDate string                   `json:"receipt_date"`
	Notes       string                   `json:"notes"`
	Items       []CreateReceiptItemInput `json:"items" binding:"required,min=1"`
}

type UpdateReceiptRequest struct {
	Number      string  `json:"number"`
	SupplierID  *string `json:"supplier_id"`
	ReceiptDate string  `json:"receipt_date"`
	Notes       string  `json:"notes"`
}

type AddReceiptItemRequest struct {
	ItemID   string  `json:"item_id"  binding:"required"`
	Quantity float64 `json:"quantity" binding:"required,gt=0"`
	Price    float64 `json:"price"    binding:"gte=0"`
	Notes    string  `json:"notes"`
}

// ─── Платежи по накладным ─────────────────────────────────────

type ReceiptPayment struct {
	ID                string  `db:"id"                  json:"id"`
	ReceiptID         string  `db:"receipt_id"          json:"receipt_id"`
	SupplierID        *string `db:"supplier_id"         json:"supplier_id"`
	Amount            float64 `db:"amount"              json:"amount"`
	PaymentMethod     string  `db:"payment_method"      json:"payment_method"`
	PaidAt            string  `db:"paid_at"             json:"paid_at"`
	Notes             string  `db:"notes"               json:"notes"`
	IsSupplierPayment bool    `db:"is_supplier_payment" json:"is_supplier_payment"`
	CreatedBy         *string `db:"created_by"          json:"created_by"`
	CreatedAt         string  `db:"created_at"          json:"created_at"`
}

type CreateReceiptPaymentRequest struct {
	Amount        float64 `json:"amount"         binding:"required,gt=0"`
	PaymentMethod string  `json:"payment_method"`
	PaidAt        string  `json:"paid_at"`
	Notes         string  `json:"notes"`
}