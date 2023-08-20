package s3manager

import (
	"github.com/gofiber/fiber/v2"
)

// HandleBucketsView renders all buckets on an HTML page.
func (s *S3Manager) HandleBucketsView(c *fiber.Ctx) error {
	return c.Render("buckets", fiber.Map{})
}
