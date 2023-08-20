package s3manager

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
)

// HandleCreateBucket creates a new bucket.
func (s *S3Manager) HandleCreateBucket(c *fiber.Ctx) error {
	name := c.FormValue("name")
	if strings.TrimSpace(name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	err := s.s3.MakeBucket(c.Context(), name, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("error making bucket: %w", err)
	}

	c.Response().Header.Set("HX-Trigger", "bucketListChanged")
	return c.SendStatus(fiber.StatusCreated)
}
