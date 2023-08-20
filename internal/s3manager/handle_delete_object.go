package s3manager

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
)

// HandleDeleteObject deletes an object.
func (s *S3Manager) HandleDeleteObject(c *fiber.Ctx) error {
	bucketName := c.Params("bucket")
	objectName := c.Params("+")

	err := s.s3.RemoveObject(c.Context(), bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("error removing object: %w", err)
	}

	c.Response().Header.Set("HX-Location", "/buckets/"+bucketName)
	return c.SendStatus(fiber.StatusNoContent)
}
