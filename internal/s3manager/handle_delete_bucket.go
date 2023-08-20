package s3manager

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// HandleDeleteBucket deletes a bucket.
func (s *S3Manager) HandleDeleteBucket(c *fiber.Ctx) error {
	bucket := c.Params("bucket")

	err := s.s3.RemoveBucket(c.Context(), bucket)
	if err != nil {
		return fmt.Errorf("error removing bucket: %w", err)
	}

	c.Response().Header.Set("HX-Location", "/buckets")
	return c.SendStatus(fiber.StatusNoContent)
}
