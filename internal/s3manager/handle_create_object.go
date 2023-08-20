package s3manager

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
)

// HandleCreateObject creates a new object.
func (s *S3Manager) HandleCreateObject(c *fiber.Ctx) error {
	bucketName := c.Params("bucket")

	form, err := c.MultipartForm()
	if err != nil {
		return fmt.Errorf("error parsing multipart form: %w", err)
	}

	pathValues := form.Value["path"]
	if len(pathValues) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "missing path form value")
	}
	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no files in form")
	}

	opts := minio.PutObjectOptions{ContentType: "application/octet-stream"}
	for _, fileHeader := range fileHeaders {
		objectName := pathValues[0] + fileHeader.Filename
		file, err := fileHeader.Open()
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", fileHeader.Filename, err)
		}
		_, err = s.s3.PutObject(c.Context(), bucketName, objectName, file, fileHeader.Size, opts)
		if err != nil {
			return fmt.Errorf("error putting object: %w", err)
		}
	}

	c.Response().Header.Set("HX-Trigger", "objectListChanged")
	return c.SendStatus(fiber.StatusCreated)
}
