package uploader

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rahadianap/community-waste-api/internal/domain"
)

type FileUploader interface {
	UploadProof(c *fiber.Ctx, file *multipart.FileHeader) (string, error)
}

type LocalUploader struct {
	UploadDir       string
	MaxUploadSizeMB int
}

func NewLocalUploader(uploadDir string, maxUploadSizeMB int) *LocalUploader {
	return &LocalUploader{
		UploadDir:       uploadDir,
		MaxUploadSizeMB: maxUploadSizeMB,
	}
}

var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".pdf":  true,
}

func (u *LocalUploader) UploadProof(c *fiber.Ctx, file *multipart.FileHeader) (string, error) {
	if file == nil {
		return "", domain.ErrProofFileRequired
	}

	maxBytes := int64(u.MaxUploadSizeMB) * 1024 * 1024
	if file.Size > maxBytes {
		return "", domain.ErrFileSizeExceeded
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return "", domain.ErrInvalidFileType
	}

	if err := os.MkdirAll(u.UploadDir, 0755); err != nil {
		return "", fmt.Errorf("gagal membuat direktori upload: %w", err)
	}

	uniqueID := uuid.New().String()
	sanitizedName := fmt.Sprintf("proof_%s%s", uniqueID, ext)
	destination := filepath.Join(u.UploadDir, sanitizedName)

	if err := c.SaveFile(file, destination); err != nil {
		return "", fmt.Errorf("gagal menyimpan file: %w", err)
	}

	publicURL := fmt.Sprintf("/uploads/payments/%s", sanitizedName)
	return publicURL, nil
}
