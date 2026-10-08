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

	// 1. Check size
	maxBytes := int64(u.MaxUploadSizeMB) * 1024 * 1024
	if file.Size > maxBytes {
		return "", domain.ErrFileSizeExceeded
	}

	// 2. Check extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExtensions[ext] {
		return "", domain.ErrInvalidFileType
	}

	// 3. Ensure target directory exists
	if err := os.MkdirAll(u.UploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// 4. Generate unique filename
	uniqueID := uuid.New().String()
	sanitizedName := fmt.Sprintf("proof_%s%s", uniqueID, ext)
	destination := filepath.Join(u.UploadDir, sanitizedName)

	// 5. Save file using Fiber's SaveFile
	if err := c.SaveFile(file, destination); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Return public/relative path: e.g. /uploads/payments/proof_123.jpg
	publicURL := fmt.Sprintf("/uploads/payments/%s", sanitizedName)
	return publicURL, nil
}
