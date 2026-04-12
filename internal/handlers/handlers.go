package handlers

import (
	"database/sql"
	"net/http"

	"jevon/internal/auth"
	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// ── Auth ──────────────────────────────────────────────────────

type AuthHandler struct {
	users   *repository.UserRepo
	authSvc *auth.Service
}

func NewAuthHandler(users *repository.UserRepo, authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{users: users, authSvc: authSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	login := req.Phone
	if login == "" {
		login = req.Email
	}
	if login == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Введите номер телефона"})
		return
	}

	user, err := h.users.FindByLogin(c, login)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный логин или пароль"})
		return
	}

	accessToken, err := h.authSvc.GenerateAccessToken(user.ID, user.Email, user.RoleName, user.RoleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}
	refreshToken, err := h.authSvc.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}
	h.users.StoreRefreshToken(c, user.ID, refreshToken, "7 days")
	c.JSON(http.StatusOK, models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := h.authSvc.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	valid, _ := h.users.ValidateRefreshToken(c, userID, req.RefreshToken)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh token not found"})
		return
	}
	user, _ := h.users.FindByID(c, userID)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	accessToken, _ := h.authSvc.GenerateAccessToken(user.ID, user.Email, user.RoleName, user.RoleID)
	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	c.ShouldBindJSON(&req)
	userID, _ := h.authSvc.ParseRefreshToken(req.RefreshToken)
	h.users.DeleteRefreshTokens(c, userID)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// ── Users / Employees ─────────────────────────────────────────

type UsersHandler struct {
	repo *repository.UserRepo
}

func NewUsersHandler(repo *repository.UserRepo) *UsersHandler {
	return &UsersHandler{repo: repo}
}

func (h *UsersHandler) List(c *gin.Context) {
	users, err := h.repo.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *UsersHandler) Get(c *gin.Context) {
	user, err := h.repo.FindByID(c, c.Param("id"))
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UsersHandler) Create(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.Create(c, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "телефон уже занят или неверная роль"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *UsersHandler) Update(c *gin.Context) {
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Update(c, c.Param("id"), req); err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "обновлено"})
}

func (h *UsersHandler) ToggleActive(c *gin.Context) {
	isActive, err := h.repo.ToggleActive(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_active": isActive})
}

func (h *UsersHandler) RoleList(c *gin.Context) {
	roles, err := h.repo.RoleList(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if roles == nil {
		roles = []models.Role{}
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func (h *UsersHandler) Assignable(c *gin.Context) {
	users, err := h.repo.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type AssignableUser struct {
		ID        string `json:"id"`
		FullName  string `json:"full_name"`
		LastName  string `json:"last_name"`
		RoleName  string `json:"role_name"`
		AvatarURL string `json:"avatar_url"`
		IsActive  bool   `json:"is_active"`
	}
	result := []AssignableUser{}
	for _, u := range users {
		if u.IsActive {
			result = append(result, AssignableUser{
				ID:        u.ID,
				FullName:  u.FullName,
				LastName:  u.LastName,
				RoleName:  u.RoleName,
				AvatarURL: u.AvatarURL,
				IsActive:  u.IsActive,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ── Dashboard ─────────────────────────────────────────────────

type DashboardHandler struct {
	repo *repository.DashboardRepo
}

func NewDashboardHandler(repo *repository.DashboardRepo) *DashboardHandler {
	return &DashboardHandler{repo: repo}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.repo.Stats(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ── Projects ──────────────────────────────────────────────────

type ProjectsHandler struct {
	repo *repository.ProjectRepo
}

func NewProjectsHandler(repo *repository.ProjectRepo) *ProjectsHandler {
	return &ProjectsHandler{repo: repo}
}

func (h *ProjectsHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	projects, err := h.repo.List(c, claims.UserID, claims.RoleName, c.Query("status"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if projects == nil {
		projects = []models.Project{}
	}
	c.JSON(http.StatusOK, gin.H{"data": projects})
}

func (h *ProjectsHandler) Get(c *gin.Context) {
	p, err := h.repo.GetByID(c, c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "не найдено"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *ProjectsHandler) Create(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateProjectRequest
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

func (h *ProjectsHandler) Update(c *gin.Context) {
	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Update(c, c.Param("project_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *ProjectsHandler) Delete(c *gin.Context) {
	h.repo.SoftDelete(c, c.Param("project_id"))
	c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
}

func (h *ProjectsHandler) GetOrders(c *gin.Context) {
	orders, err := h.repo.GetProjectOrders(c, c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if orders == nil {
		orders = []models.ProjectOrder{}
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *ProjectsHandler) AddOrder(c *gin.Context) {
	var req struct {
		OrderID string `json:"order_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.AddOrderToProject(c, c.Param("project_id"), req.OrderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "добавлено"})
}

func (h *ProjectsHandler) RemoveOrder(c *gin.Context) {
	if err := h.repo.RemoveOrderFromProject(c, c.Param("project_id"), c.Param("order_id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "удалено"})
}

// ── Tasks ─────────────────────────────────────────────────────

type TasksHandler struct {
	repo *repository.TaskRepo
}

func NewTasksHandler(repo *repository.TaskRepo) *TasksHandler {
	return &TasksHandler{repo: repo}
}

func (h *TasksHandler) List(c *gin.Context) {
	claims := middleware.GetClaims(c)
	tasks, err := h.repo.List(c,
		claims.UserID, claims.RoleName,
		c.Query("project_id"), c.Query("assigned_to"), c.Query("status"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

func (h *TasksHandler) Create(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.Create(c, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *TasksHandler) Update(c *gin.Context) {
	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Update(c, c.Param("id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *TasksHandler) UpdateStatus(c *gin.Context) {
	var req models.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.repo.UpdateStatus(c, c.Param("id"), req.Status)
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

func (h *TasksHandler) Delete(c *gin.Context) {
	h.repo.Delete(c, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

var _ = sql.ErrNoRows