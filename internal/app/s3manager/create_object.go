package s3manager

import (
	"fmt"
	"net/http"

	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
)

// HandleCreateObject uploads a new object.
func HandleCreateObject(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")

		err := r.ParseMultipartForm(0)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing multipart form: %w", err))
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting file from form: %w", err))
			return
		}
		defer file.Close()

		opts := minio.PutObjectOptions{ContentType: "application/octet-stream"}
		_, err = s3.PutObject(bucketName, header.Filename, file, -1, opts)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error putting object: %w", err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
