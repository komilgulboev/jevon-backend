package storage

import "jevon/internal/models"

// ToCreateFileRequest конвертирует UploadedFile в models.CreateFileRequest
func ToCreateFileRequest(f UploadedFile) models.CreateFileRequest {
	return models.CreateFileRequest{
		FileName: f.FileName,
		FileURL:  f.FileURL,
		FileType: f.FileType,
		FileSize: f.FileSize,
	}
}
