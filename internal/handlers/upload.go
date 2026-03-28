package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"jevon/internal/middleware"
	"jevon/internal/models"
	"jevon/internal/repository"
	"jevon/internal/storage"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	storage   *storage.MinIOService
	pipeline  *repository.PipelineRepo
	orderRepo *repository.OrderRepo
}

func NewUploadHandler(s *storage.MinIOService, pipeline *repository.PipelineRepo) *UploadHandler {
	return &UploadHandler{storage: s, pipeline: pipeline}
}

func (h *UploadHandler) SetOrderRepo(repo *repository.OrderRepo) {
	h.orderRepo = repo
}

// POST /api/projects/:project_id/stages/:stage_id/upload
// POST /api/orders/:order_id/stages/:stage_id/upload
// Query params: type=project, category=preliminary|design|drawing|finished|installation|handover|other
func (h *UploadHandler) UploadStageFiles(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "file storage not available"})
		return
	}

	claims     := middleware.GetClaims(c)
	stageID    := c.Param("stage_id")
	uploadType := c.DefaultQuery("type", "project")
	category   := c.DefaultQuery("category", "other")

	projectID := c.Param("project_id")
	orderID   := c.Param("order_id")
	if projectID == "" {
		projectID = orderID
	}
	isOrder := orderID != ""

	form, err := c.MultipartForm()
	if err != nil {
		log.Printf("❌ MultipartForm error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form: " + err.Error()})
		return
	}

	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		log.Printf("❌ No files in form, keys: %v", func() []string {
			keys := make([]string, 0)
			for k := range form.File {
				keys = append(keys, k)
			}
			return keys
		}())
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files in form field 'files'"})
		return
	}

	log.Printf("📁 Uploading %d file(s), type=%s, category=%s, id=%s",
		len(fileHeaders), uploadType, category, projectID)

	var uploadedFiles []storage.UploadedFile
	for _, header := range fileHeaders {
		file, err := header.Open()
		if err != nil {
			log.Printf("❌ Open file error [%s]: %v", header.Filename, err)
			continue
		}
		defer file.Close()

		fileURL, fileName, err := h.storage.Upload(c, file, header, uploadType, projectID)
		if err != nil {
			log.Printf("❌ MinIO upload error [%s]: %v", header.Filename, err)
			continue
		}

		log.Printf("✅ Uploaded: %s → %s", fileName, fileURL)

		uploadedFiles = append(uploadedFiles, storage.UploadedFile{
			FileName: fileName,
			FileURL:  fileURL,
			FileType: header.Header.Get("Content-Type"),
			FileSize: header.Size,
		})
	}

	if len(uploadedFiles) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload files to storage"})
		return
	}

	var savedIDs []string
	for _, f := range uploadedFiles {
		req := storage.ToCreateFileRequest(f)
		// Добавляем категорию
		modelReq := models.CreateFileRequest{
			FileName: req.FileName,
			FileURL:  req.FileURL,
			FileType: req.FileType,
			FileSize: req.FileSize,
			Category: category,
		}
		id, err := h.pipeline.CreateFile(c, projectID, stageID, claims.UserID, modelReq)
		if err != nil {
			log.Printf("❌ DB save error: %v", err)
		} else {
			savedIDs = append(savedIDs, id)
		}
	}

	// Логируем в историю заказа
	if isOrder && h.orderRepo != nil && len(savedIDs) > 0 {
		names := make([]string, 0, len(uploadedFiles))
		for _, f := range uploadedFiles {
			names = append(names, f.FileName)
		}
		categoryLabel := fileCategoryLabel(category)
		comment := fmt.Sprintf("📎 Загружено файлов: %d [%s] | %s",
			len(savedIDs), categoryLabel, strings.Join(names, ", "))
		h.orderRepo.LogHistory(c, orderID, "files", "files", claims.UserID, comment)
	}

	c.JSON(http.StatusCreated, gin.H{
		"uploaded": len(uploadedFiles),
		"files":    uploadedFiles,
		"ids":      savedIDs,
	})
}

// POST /api/users/avatar
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "file storage not available"})
		return
	}

	claims := middleware.GetClaims(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	fileURL, _, err := h.storage.Upload(c, file, header, "avatar", claims.UserID)
	if err != nil {
		log.Printf("❌ Avatar upload error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"avatar_url": fileURL})
}

// DELETE /api/files
func (h *UploadHandler) DeleteFile(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "file storage not available"})
		return
	}

	var req struct {
		FileURL string `json:"file_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.storage.Delete(c, req.FileURL); err != nil {
		log.Printf("❌ Delete file error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// helper
func fileCategoryLabel(category string) string {
	labels := map[string]string{
		"preliminary":  "Предварительные фото",
		"design":       "Дизайн",
		"drawing":      "Чертёж",
		"finished":     "Готовые работы",
		"installation": "Установка",
		"handover":     "Сдача",
		"other":        "Другое",
	}
	if l, ok := labels[category]; ok {
		return l
	}
	return category
}