package models

import "time"

// ─── Проект ───────────────────────────────────────────────

type Project struct {
	ID            string `json:"id"`
	ProjectNumber int    `json:"project_number"`
	Title         string `json:"title"`
	ClientID      string `json:"client_id"`
	ClientName    string `json:"client_name"`
	ClientPhone   string `json:"client_phone"`
	Status        string `json:"status"`
	Priority      string `json:"priority"`
	Deadline      string `json:"deadline"`
	CreatedBy     string `json:"created_by"`
	CreatedAt     string `json:"created_at"`
	Notes         string `json:"notes"`
	OrderCount    int    `json:"order_count"`
	// Старые поля — оставляем для совместимости
	CurrentStage string `json:"current_stage"`
	Progress     int    `json:"progress"`
	TotalTasks   int    `json:"total_tasks"`
}

type ProjectOrder struct {
	ID            string  `json:"id"`
	OrderNumber   int     `json:"order_number"`
	OrderType     string  `json:"order_type"`
	Title         string  `json:"title"`
	ClientName    string  `json:"client_name"`
	ClientPhone   string  `json:"client_phone"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"payment_status"`
	FinalCost     float64 `json:"final_cost"`
	PaidAmount    float64 `json:"paid_amount"`
	Deadline      string  `json:"deadline"`
	CurrentStage  string  `json:"current_stage"`
	Progress      int     `json:"progress"`
	TotalStages   int     `json:"total_stages"`
	DoneStages    int     `json:"done_stages"`
}

type CreateProjectRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	ClientID    string   `json:"client_id"`
	ClientName  string   `json:"client_name"`
	ClientPhone string   `json:"client_phone"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Deadline    string   `json:"deadline"`
	Notes       string   `json:"notes"`
	OrderIDs    []string `json:"order_ids"`
	MemberIDs   []string `json:"member_ids"`
}

type UpdateProjectRequest struct {
	Title       string `json:"title"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Deadline    string `json:"deadline"`
	ClientName  string `json:"client_name"`
	ClientPhone string `json:"client_phone"`
	Notes       string `json:"notes"`
}

// ─── Задача ───────────────────────────────────────────────

type Task struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	ProjectTitle   string    `json:"project_title"`
	AssignedTo     string    `json:"assigned_to"`
	AssignedToName string    `json:"assigned_to_name"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	Priority       string    `json:"priority"`
	DueDate        *string   `json:"due_date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateTaskRequest struct {
	ProjectID   string  `json:"project_id" binding:"required"`
	Title       string  `json:"title"      binding:"required"`
	Description string  `json:"description"`
	AssignedTo  string  `json:"assigned_to"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssignedTo  *string `json:"assigned_to"`
	DueDate     *string `json:"due_date"`
}

type UpdateTaskStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ─── Dashboard ────────────────────────────────────────────

type DashboardStats struct {
	ActiveProjects  int `json:"active_projects"`
	TasksInProgress int `json:"tasks_in_progress"`
	TasksDone       int `json:"tasks_done"`
	TotalEmployees  int `json:"total_employees"`
	ProjectsDone    int `json:"projects_done"`
	TasksOverdue    int `json:"tasks_overdue"`
}