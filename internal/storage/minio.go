package storage

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"jevon/internal/config"
)

type MinIOService struct {
	client *minio.Client
	cfg    config.MinIOConfig
}

func NewMinIOService(cfg config.MinIOConfig) (*MinIOService, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio connect: %w", err)
	}
	return &MinIOService{client: client, cfg: cfg}, nil
}

// BucketByType возвращает бакет по типу загрузки
func (s *MinIOService) BucketByType(uploadType string) string {
	switch uploadType {
	case "design":
		return s.cfg.BucketDesign
	case "cutting":
		return s.cfg.BucketCutting
	case "avatar":
		return s.cfg.BucketAvatars
	default:
		return s.cfg.BucketProjects
	}
}

// Upload загружает файл в MinIO и возвращает публичный URL
func (s *MinIOService) Upload(
	ctx context.Context,
	file multipart.File,
	header *multipart.FileHeader,
	uploadType string, // "project", "design", "cutting", "avatar"
	folder string,     // projectID или userID
) (fileURL, fileName string, err error) {

	bucket := s.BucketByType(uploadType)

	// Генерируем уникальное имя файла
	ext := strings.ToLower(filepath.Ext(header.Filename))
	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	// Определяем content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Загружаем в MinIO
	_, err = s.client.PutObject(ctx, bucket, objectName, file, header.Size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", "", fmt.Errorf("upload error: %w", err)
	}

	// Формируем публичный URL
	scheme := "http"
	if s.cfg.UseSSL {
		scheme = "https"
	}
	fileURL = fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.Endpoint, bucket, objectName)

	return fileURL, header.Filename, nil
}

// UploadMultiple загружает несколько файлов
func (s *MinIOService) UploadMultiple(
	ctx context.Context,
	form *multipart.Form,
	uploadType string,
	folder string,
) ([]UploadedFile, error) {
	files := form.File["files"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	var result []UploadedFile
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		url, name, err := s.Upload(ctx, file, header, uploadType, folder)
		if err != nil {
			continue
		}
		result = append(result, UploadedFile{
			FileName: name,
			FileURL:  url,
			FileType: header.Header.Get("Content-Type"),
			FileSize: header.Size,
		})
	}
	return result, nil
}

// Delete удаляет файл из MinIO
func (s *MinIOService) Delete(ctx context.Context, fileURL string) error {
	// Извлекаем bucket и objectName из URL
	// URL формат: http://endpoint/bucket/folder/uuid.ext
	parts := strings.SplitN(fileURL, "/", 5)
	if len(parts) < 5 {
		return fmt.Errorf("invalid file URL: %s", fileURL)
	}
	bucket := parts[3]
	objectName := parts[4]

	return s.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
}

// PresignedURL генерирует временный URL для приватного доступа
func (s *MinIOService) PresignedURL(ctx context.Context, fileURL string, expiry time.Duration) (string, error) {
	parts := strings.SplitN(fileURL, "/", 5)
	if len(parts) < 5 {
		return "", fmt.Errorf("invalid file URL")
	}
	bucket := parts[3]
	objectName := parts[4]

	url, err := s.client.PresignedGetObject(ctx, bucket, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// UploadedFile результат загрузки файла
type UploadedFile struct {
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
}
