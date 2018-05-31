package s3manager

import (
	"fmt"
	"io"
	"net/http"

	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// HandleGetObject downloads an object to the client.
func HandleGetObject(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")
		objectName := way.Param(r.Context(), "objectName")

		object, err := s3.GetObject(bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error getting object"))
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", objectName))
		w.Header().Set("Content-Type", "application/octet-stream")
		_, err = io.Copy(w, object)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error copying object to response writer"))
			return
		}
	}
}
