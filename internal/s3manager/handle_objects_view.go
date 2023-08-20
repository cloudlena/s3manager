package s3manager

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

func (s *S3Manager) HandleObjects(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	path := c.Params("*")

	return c.Render("objects", fiber.Map{
		"BucketName": bucket,
		"Path":       path,
		"PathParts":  removeEmptyStrings(strings.Split(path, "/")),
	})
}

func removeEmptyStrings(input []string) []string {
	var result []string
	for _, str := range input {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}
