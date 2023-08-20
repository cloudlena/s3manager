package s3manager

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
)

type objectWithIcon struct {
	Key          string
	Icon         string
	DisplayName  string
	LastModified time.Time
	StorageClass string
	Size         int64
	IsFolder     bool
}

func (s *S3Manager) HandleObjectList(c *fiber.Ctx) error {
	bucket := c.Params("bucket")
	path := c.Params("*")

	if path != "" {
		path += "/"
	}

	var objects []objectWithIcon
	doneCh := make(chan struct{})
	defer close(doneCh)
	opts := minio.ListObjectsOptions{
		Prefix: path,
	}
	objectCh := s.s3.ListObjects(c.Context(), bucket, opts)
	for object := range objectCh {
		if object.Err != nil {
			if strings.Contains(object.Err.Error(), "does not exist") {
				return fiber.NewError(fiber.StatusNotFound, object.Err.Error())
			}
			return fmt.Errorf("error listing objects: %w", object.Err)
		}

		o := objectWithIcon{
			Key:          object.Key,
			Icon:         icon(object.Key),
			DisplayName:  strings.TrimPrefix(object.Key, path),
			Size:         object.Size,
			LastModified: object.LastModified,
			StorageClass: object.StorageClass,
			IsFolder:     strings.HasSuffix(object.Key, "/"),
		}
		objects = append(objects, o)
	}

	return c.Render("object-list", fiber.Map{
		"BucketName":  bucket,
		"Objects":     objects,
		"AllowDelete": s.allowDelete,
		"Path":        path,
	})
}

// icon returns an icon for a file type.
func icon(fileName string) string {
	if strings.HasSuffix(fileName, "/") {
		return "folder"
	}

	e := path.Ext(fileName)
	switch e {
	case ".tgz", ".gz", ".zip":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3", ".wav":
		return "music_note"
	}

	return "insert_drive_file"
}
