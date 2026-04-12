package models

import "time"

// ─── Роли ────────────────────────────────────────────────

type Role struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ─── Пользователь / Сотрудник ────────────────────────────

type User struct {
	ID           string    `json:"id"`
	RoleID       int       `json:"role_id"`
	RoleName     string    `json:"role_name"`
	FullName     string    `json:"full_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Phone        string    `json:"phone"`
	WhatsApp     string    `json:"whatsapp"`
	Telegram     string    `json:"telegram"`
	Address      string    `json:"address"`
	Salary       *float64  `json:"salary"`
	HourlyRate   *float64  `json:"hourly_rate"`
	ContractType string    `json:"contract_type"`
	IsActive     bool      `json:"is_active"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ─── Auth DTOs ────────────────────────────────────────────

type LoginRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ─── User DTOs ────────────────────────────────────────────

type CreateUserRequest struct {
	RoleID       int      `json:"role_id"   binding:"required"`
	FullName     string   `json:"full_name" binding:"required"`
	LastName     string   `json:"last_name"`
	Phone        string   `json:"phone"     binding:"required"`
	Email        string   `json:"email"`
	Password     string   `json:"password"  binding:"required,min=6"`
	WhatsApp     string   `json:"whatsapp"`
	Telegram     string   `json:"telegram"`
	Address      string   `json:"address"`
	Salary       *float64 `json:"salary"`
	HourlyRate   *float64 `json:"hourly_rate"`
	ContractType string   `json:"contract_type"`
}

type UpdateUserRequest struct {
	RoleID       *int     `json:"role_id"`
	FullName     *string  `json:"full_name"`
	LastName     *string  `json:"last_name"`
	Phone        *string  `json:"phone"`
	Email        *string  `json:"email"`
	Password     string   `json:"password"` // пустой = не менять
	WhatsApp     *string  `json:"whatsapp"`
	Telegram     *string  `json:"telegram"`
	Address      *string  `json:"address"`
	Salary       *float64 `json:"salary"`
	HourlyRate   *float64 `json:"hourly_rate"`
	ContractType *string  `json:"contract_type"`
}