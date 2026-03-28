package models

import "time"

// ── Operation Catalog ─────────────────────────────────────

type OperationCatalog struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsActive    bool   `json:"is_active"`
}

// ── Project Stage ─────────────────────────────────────────

type ProjectStage struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	Stage        string     `json:"stage"`
	Status       string     `json:"status"`
	AssignedTo   string     `json:"assigned_to"`
	AssigneeName string     `json:"assignee_name"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Notes        string     `json:"notes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	Operations []StageOperation `json:"operations,omitempty"`
	Files      []StageFile      `json:"files,omitempty"`
}

// ── Stage Operation ───────────────────────────────────────

type StageOperation struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	StageID      string     `json:"stage_id"`
	CatalogID    *int       `json:"catalog_id"`
	CatalogName  string     `json:"catalog_name"`
	CustomName   string     `json:"custom_name"`
	AssignedTo   string     `json:"assigned_to"`
	AssigneeName string     `json:"assignee_name"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Notes        string     `json:"notes"`
	CreatedAt    time.Time  `json:"created_at"`

	Materials []OperationMaterial `json:"materials,omitempty"`
}

func (o StageOperation) OperationName() string {
	if o.CustomName != "" {
		return o.CustomName
	}
	return o.CatalogName
}

// ── Operation Material ────────────────────────────────────

type OperationMaterial struct {
	ID            string    `json:"id"`
	OperationID   string    `json:"operation_id"`
	ProjectID     string    `json:"project_id"`
	Name          string    `json:"name"`
	Quantity      float64   `json:"quantity"`
	Unit          string    `json:"unit"`
	UnitPrice     float64   `json:"unit_price"`
	TotalPrice    float64   `json:"total_price"`
	Supplier      string    `json:"supplier"`
	SupplierPhone string    `json:"supplier_phone"`
	PurchasedAt   *string   `json:"purchased_at"`
	Notes         string    `json:"notes"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// ── Stage File ────────────────────────────────────────────

type StageFile struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	StageID    string    `json:"stage_id"`
	FileName   string    `json:"file_name"`
	FileURL    string    `json:"file_url"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	UploadedBy string    `json:"uploaded_by"`
	Category   string    `json:"category"`
	CreatedAt  time.Time `json:"created_at"`
}

// ── Project History ───────────────────────────────────────

type ProjectHistory struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	FromStage   string    `json:"from_stage"`
	ToStage     string    `json:"to_stage"`
	ChangedBy   string    `json:"changed_by"`
	ChangerName string    `json:"changer_name"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
}

// ── Request DTOs ──────────────────────────────────────────

type UpdateStageRequest struct {
	Status     *string `json:"status"`
	AssignedTo *string `json:"assigned_to"`
	Notes      *string `json:"notes"`
}

type CompleteStageRequest struct {
	Notes   string `json:"notes"`
	Comment string `json:"comment"`
}

type CreateOperationRequest struct {
	StageID    string `json:"stage_id"    binding:"required"`
	CatalogID  *int   `json:"catalog_id"`
	CustomName string `json:"custom_name"`
	AssignedTo string `json:"assigned_to"`
	Notes      string `json:"notes"`
}

type UpdateOperationRequest struct {
	Status     *string `json:"status"`
	AssignedTo *string `json:"assigned_to"`
	Notes      *string `json:"notes"`
}

type CreateMaterialRequest struct {
	Name          string  `json:"name"         binding:"required"`
	Quantity      float64 `json:"quantity"     binding:"required"`
	Unit          string  `json:"unit"`
	UnitPrice     float64 `json:"unit_price"`
	Supplier      string  `json:"supplier"`
	SupplierPhone string  `json:"supplier_phone"`
	PurchasedAt   *string `json:"purchased_at"`
	Notes         string  `json:"notes"`
}

type CreateFileRequest struct {
	FileName string `json:"file_name" binding:"required"`
	FileURL  string `json:"file_url"  binding:"required"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
	Category string `json:"category"`
}

// ── Stage labels ──────────────────────────────────────────

var StageLabels = map[string]string{
	"intake":     "Приём заказа",
	"design":     "Дизайн",
	"cutting":    "Раскрой",
	"production": "Производство",
	"warehouse":  "Склад",
	"delivery":   "Доставка",
	"assembly":   "Сборка",
	"handover":   "Сдача",
}

var StageOrder = []string{
	"intake", "design", "cutting", "production",
	"warehouse", "delivery", "assembly", "handover",
}

// ── File categories ───────────────────────────────────────

var FileCategories = []struct {
	Key   string
	Label string
}{
	{"preliminary", "Предварительные фото"},
	{"design",      "Дизайн"},
	{"drawing",     "Чертёж"},
	{"finished",    "Готовые работы"},
	{"installation","Установка"},
	{"handover",    "Сдача"},
	{"other",       "Другое"},
}