package models

import "time"

// ─── Проект ───────────────────────────────────────────────

type Project struct {
	ID            string    `json:"id"`
	ProjectNumber int       `json:"project_number"`
	CurrentStage  string    `json:"current_stage"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ClientName    string    `json:"client_name"`
	ClientPhone   string    `json:"client_phone"`
	Status        string    `json:"status"`
	Priority      string    `json:"priority"`
	Deadline      *string   `json:"deadline"`
	CreatedBy     string    `json:"created_by"`
	Progress      int       `json:"progress"`
	TotalTasks    int       `json:"total_tasks"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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

// ─── Request DTOs ─────────────────────────────────────────

type CreateProjectRequest struct {
	Title       string   `json:"title"       binding:"required"`
	Description string   `json:"description"`
	ClientName  string   `json:"client_name"`
	ClientPhone string   `json:"client_phone"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Deadline    *string  `json:"deadline"`
	MemberIDs   []string `json:"member_ids"`
}

type UpdateProjectRequest struct {
	Title       *string `json:"title"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	Deadline    *string `json:"deadline"`
	ClientName  *string `json:"client_name"`
	ClientPhone *string `json:"client_phone"`
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