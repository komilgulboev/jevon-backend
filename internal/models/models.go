package models

import "time"

type Role struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type User struct {
	ID           string    `json:"id"`
	RoleID       int       `json:"role_id"`
	RoleName     string    `json:"role_name"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Phone        string    `json:"phone"`
	IsActive     bool      `json:"is_active"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

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

// ── Request DTOs ──────────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type CreateUserRequest struct {
	RoleID   int    `json:"role_id"   binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email"     binding:"required,email"`
	Password string `json:"password"  binding:"required,min=6"`
	Phone    string `json:"phone"`
}

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

type DashboardStats struct {
	ActiveProjects  int `json:"active_projects"`
	TasksInProgress int `json:"tasks_in_progress"`
	TasksDone       int `json:"tasks_done"`
	TotalEmployees  int `json:"total_employees"`
	ProjectsDone    int `json:"projects_done"`
	TasksOverdue    int `json:"tasks_overdue"`
}