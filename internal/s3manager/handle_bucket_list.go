package s3manager

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// HandleBucketList renders all buckets as an HTML list.
func (s *S3Manager) HandleBucketList(c *fiber.Ctx) error {
	buckets, err := s.s3.ListBuckets(c.Context())
	if err != nil {
		return fmt.Errorf("error listing buckets: %w", err)
	}

	return c.Render("bucket-list", fiber.Map{
		"Buckets": buckets,
	})
}
