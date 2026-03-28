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

// ── Auth ──────────────────────────────────────────────────

type AuthHandler struct {
	users   *repository.UserRepo
	authSvc *auth.Service
}

func NewAuthHandler(users *repository.UserRepo, authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{users: users, authSvc: authSvc}
}

// @Summary Login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body models.LoginRequest true "Credentials"
// @Success 200 {object} models.LoginResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := h.users.FindByEmail(c, req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
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

// @Summary Refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body models.RefreshRequest true "Refresh token"
// @Success 200 {object} map[string]string
// @Router /auth/refresh [post]
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

// @Summary Logout
// @Tags auth
// @Security BearerAuth
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.RefreshRequest
	c.ShouldBindJSON(&req)
	userID, _ := h.authSvc.ParseRefreshToken(req.RefreshToken)
	h.users.DeleteRefreshTokens(c, userID)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

// ── Users ─────────────────────────────────────────────────

type UsersHandler struct {
	repo *repository.UserRepo
}

func NewUsersHandler(repo *repository.UserRepo) *UsersHandler {
	return &UsersHandler{repo: repo}
}

// @Summary List users
// @Tags users
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /users [get]
func (h *UsersHandler) List(c *gin.Context) {
	users, err := h.repo.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// @Summary Create user
// @Tags users
// @Security BearerAuth
// @Param body body models.CreateUserRequest true "User data"
// @Router /users [post]
func (h *UsersHandler) Create(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.Create(c, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email already exists or invalid role"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary Get user
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Router /users/{id} [get]
func (h *UsersHandler) Get(c *gin.Context) {
	user, err := h.repo.FindByID(c, c.Param("id"))
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// @Summary Toggle user active status
// @Tags users
// @Security BearerAuth
// @Param id path string true "User ID"
// @Router /users/{id}/toggle-active [patch]
func (h *UsersHandler) ToggleActive(c *gin.Context) {
	isActive, err := h.repo.ToggleActive(c, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_active": isActive})
}

// ── Dashboard ─────────────────────────────────────────────

type DashboardHandler struct {
	repo *repository.DashboardRepo
}

func NewDashboardHandler(repo *repository.DashboardRepo) *DashboardHandler {
	return &DashboardHandler{repo: repo}
}

// @Summary Dashboard statistics
// @Tags dashboard
// @Security BearerAuth
// @Success 200 {object} models.DashboardStats
// @Router /dashboard/stats [get]
func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.repo.Stats(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ── Projects ──────────────────────────────────────────────

type ProjectsHandler struct {
	repo *repository.ProjectRepo
}

func NewProjectsHandler(repo *repository.ProjectRepo) *ProjectsHandler {
	return &ProjectsHandler{repo: repo}
}

// @Summary List projects
// @Tags projects
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Router /projects [get]
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

// @Summary Create project
// @Tags projects
// @Security BearerAuth
// @Param body body models.CreateProjectRequest true "Project data"
// @Router /projects [post]
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

// @Summary Update project
// @Tags projects
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Router /projects/{project_id} [patch]
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

// @Summary Delete project
// @Tags projects
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Router /projects/{project_id} [delete]
func (h *ProjectsHandler) Delete(c *gin.Context) {
	h.repo.SoftDelete(c, c.Param("project_id"))
	c.JSON(http.StatusOK, gin.H{"message": "cancelled"})
}

// ── Tasks ─────────────────────────────────────────────────

type TasksHandler struct {
	repo *repository.TaskRepo
}

func NewTasksHandler(repo *repository.TaskRepo) *TasksHandler {
	return &TasksHandler{repo: repo}
}

// @Summary List tasks
// @Tags tasks
// @Security BearerAuth
// @Param project_id query string false "Filter by project"
// @Param status     query string false "Filter by status"
// @Router /tasks [get]
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

// @Summary Create task
// @Tags tasks
// @Security BearerAuth
// @Router /tasks [post]
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

// @Summary Update task
// @Tags tasks
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Router /tasks/{id} [patch]
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

// @Summary Update task status
// @Tags tasks
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Router /tasks/{id}/status [patch]
func (h *TasksHandler) UpdateStatus(c *gin.Context) {
	var req models.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.repo.UpdateStatus(c, c.Param("id"), req.Status)
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

// @Summary Delete task
// @Tags tasks
// @Security BearerAuth
// @Param id path string true "Task ID"
// @Router /tasks/{id} [delete]
func (h *TasksHandler) Delete(c *gin.Context) {
	h.repo.Delete(c, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Ensure sql import used
var _ = sql.ErrNoRows