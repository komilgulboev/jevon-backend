package handlers

import (
	"net/http"

	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"

	"github.com/gin-gonic/gin"
)

type PipelineHandler struct {
	repo *repository.PipelineRepo
}

func NewPipelineHandler(repo *repository.PipelineRepo) *PipelineHandler {
	return &PipelineHandler{repo: repo}
}

// ── Operation Catalog ─────────────────────────────────────

// @Summary List operation catalog
// @Tags pipeline
// @Security BearerAuth
// @Param category query string false "Filter by category"
// @Success 200 {object} map[string]interface{}
// @Router /catalog/operations [get]
func (h *PipelineHandler) CatalogList(c *gin.Context) {
	items, err := h.repo.CatalogList(c, c.Query("category"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if items == nil {
		items = []models.OperationCatalog{}
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// ── Project Stages ────────────────────────────────────────

// @Summary Get all stages of a project
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Success 200 {object} map[string]interface{}
// @Router /projects/{project_id}/stages [get]
func (h *PipelineHandler) StagesList(c *gin.Context) {
	stages, err := h.repo.StagesByProject(c, c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if stages == nil {
		stages = []models.ProjectStage{}
	}
	c.JSON(http.StatusOK, gin.H{"data": stages})
}

// @Summary Get single stage with operations and files
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Router /projects/{project_id}/stages/{stage_id} [get]
func (h *PipelineHandler) StageGet(c *gin.Context) {
	stage, err := h.repo.StageByID(c, c.Param("stage_id"))
	if err != nil || stage == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "stage not found"})
		return
	}

	// Загружаем операции
	ops, _ := h.repo.OperationsByStage(c, stage.ID)
	if ops != nil {
		// Загружаем материалы для каждой операции
		for i := range ops {
			mats, _ := h.repo.MaterialsByOperation(c, ops[i].ID)
			if mats != nil {
				ops[i].Materials = mats
			} else {
				ops[i].Materials = []models.OperationMaterial{}
			}
		}
		stage.Operations = ops
	} else {
		stage.Operations = []models.StageOperation{}
	}

	// Загружаем файлы
	files, _ := h.repo.FilesByStage(c, stage.ID)
	if files != nil {
		stage.Files = files
	} else {
		stage.Files = []models.StageFile{}
	}

	c.JSON(http.StatusOK, stage)
}

// @Summary Update stage (assign, add notes)
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Router /projects/{project_id}/stages/{stage_id} [patch]
func (h *PipelineHandler) StageUpdate(c *gin.Context) {
	var req models.UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.UpdateStage(c, c.Param("stage_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// @Summary Complete a stage — moves project to next stage
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Router /projects/{project_id}/stages/{stage_id}/complete [post]
func (h *PipelineHandler) StageComplete(c *gin.Context) {
	var req models.CompleteStageRequest
	c.ShouldBindJSON(&req)

	if err := h.repo.CompleteStage(c, c.Param("stage_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stage completed"})
}

// ── Stage Operations ──────────────────────────────────────

// @Summary List operations for a stage
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Router /projects/{project_id}/stages/{stage_id}/operations [get]
func (h *PipelineHandler) OperationsList(c *gin.Context) {
	ops, err := h.repo.OperationsByStage(c, c.Param("stage_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ops == nil {
		ops = []models.StageOperation{}
	}
	c.JSON(http.StatusOK, gin.H{"data": ops})
}

// @Summary List all operations for a project
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Router /projects/{project_id}/operations [get]
func (h *PipelineHandler) OperationsByProject(c *gin.Context) {
	ops, err := h.repo.OperationsByProject(c, c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ops == nil {
		ops = []models.StageOperation{}
	}
	c.JSON(http.StatusOK, gin.H{"data": ops})
}

// @Summary Create operation in a stage
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param body body models.CreateOperationRequest true "Operation data"
// @Router /projects/{project_id}/operations [post]
func (h *PipelineHandler) OperationCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.CatalogID == nil && req.CustomName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "catalog_id or custom_name required"})
		return
	}
	id, err := h.repo.CreateOperation(c, c.Param("project_id"), req, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary Update operation status
// @Tags pipeline
// @Security BearerAuth
// @Param project_id   path string true "Project ID"
// @Param operation_id path string true "Operation ID"
// @Router /projects/{project_id}/operations/{operation_id} [patch]
func (h *PipelineHandler) OperationUpdate(c *gin.Context) {
	var req models.UpdateOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.UpdateOperation(c, c.Param("operation_id"), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// @Summary Delete operation
// @Tags pipeline
// @Security BearerAuth
// @Param project_id   path string true "Project ID"
// @Param operation_id path string true "Operation ID"
// @Router /projects/{project_id}/operations/{operation_id} [delete]
func (h *PipelineHandler) OperationDelete(c *gin.Context) {
	h.repo.DeleteOperation(c, c.Param("operation_id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Materials ─────────────────────────────────────────────

// @Summary List materials for an operation
// @Tags pipeline
// @Security BearerAuth
// @Param project_id   path string true "Project ID"
// @Param operation_id path string true "Operation ID"
// @Router /projects/{project_id}/operations/{operation_id}/materials [get]
func (h *PipelineHandler) MaterialsList(c *gin.Context) {
	mats, err := h.repo.MaterialsByOperation(c, c.Param("operation_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if mats == nil {
		mats = []models.OperationMaterial{}
	}
	c.JSON(http.StatusOK, gin.H{"data": mats})
}

// @Summary List all materials for a project with total cost
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Router /projects/{project_id}/materials [get]
func (h *PipelineHandler) MaterialsByProject(c *gin.Context) {
	projectID := c.Param("project_id")
	mats, err := h.repo.MaterialsByProject(c, projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	total, _ := h.repo.TotalCost(c, projectID)
	if mats == nil {
		mats = []models.OperationMaterial{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":       mats,
		"total_cost": total,
	})
}

// @Summary Add material to an operation
// @Tags pipeline
// @Security BearerAuth
// @Param project_id   path string true "Project ID"
// @Param operation_id path string true "Operation ID"
// @Param body body models.CreateMaterialRequest true "Material data"
// @Router /projects/{project_id}/operations/{operation_id}/materials [post]
func (h *PipelineHandler) MaterialCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.CreateMaterial(c,
		c.Param("operation_id"),
		c.Param("project_id"),
		claims.UserID, req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary Delete material
// @Tags pipeline
// @Security BearerAuth
// @Param project_id   path string true "Project ID"
// @Param operation_id path string true "Operation ID"
// @Param material_id  path string true "Material ID"
// @Router /projects/{project_id}/operations/{operation_id}/materials/{material_id} [delete]
func (h *PipelineHandler) MaterialDelete(c *gin.Context) {
	h.repo.DeleteMaterial(c, c.Param("material_id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Stage Files ───────────────────────────────────────────

// @Summary List files for a stage
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Router /projects/{project_id}/stages/{stage_id}/files [get]
func (h *PipelineHandler) FilesList(c *gin.Context) {
	files, err := h.repo.FilesByStage(c, c.Param("stage_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if files == nil {
		files = []models.StageFile{}
	}
	c.JSON(http.StatusOK, gin.H{"data": files})
}

// @Summary Add file to a stage
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Router /projects/{project_id}/stages/{stage_id}/files [post]
func (h *PipelineHandler) FileCreate(c *gin.Context) {
	claims := middleware.GetClaims(c)
	var req models.CreateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.repo.CreateFile(c,
		c.Param("project_id"),
		c.Param("stage_id"),
		claims.UserID, req,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// @Summary Delete file
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param stage_id   path string true "Stage ID"
// @Param file_id    path string true "File ID"
// @Router /projects/{project_id}/stages/{stage_id}/files/{file_id} [delete]
func (h *PipelineHandler) FileDelete(c *gin.Context) {
	h.repo.DeleteFile(c, c.Param("file_id"))
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ── Project History ───────────────────────────────────────

// @Summary Get project stage history
// @Tags pipeline
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Router /projects/{project_id}/history [get]
func (h *PipelineHandler) History(c *gin.Context) {
	history, err := h.repo.History(c, c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if history == nil {
		history = []models.ProjectHistory{}
	}
	c.JSON(http.StatusOK, gin.H{"data": history})
}
