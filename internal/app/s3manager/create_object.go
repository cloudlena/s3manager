package s3manager

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// HandleCreateObject uploads a new object.
func HandleCreateObject(s3 S3, sseInfo SSEType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		err := r.ParseMultipartForm(32 << 20) // 32 Mb
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing multipart form: %w", err))
			return
		}
		file, _, err := r.FormFile("file")
		path := r.FormValue("path")
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting file from form: %w", err))
			return
		}
		defer func(file multipart.File) {
			if err = file.Close(); err != nil {
				log.Fatal(fmt.Errorf("file cannot be closed: %w", err))
			}
		}(file)

		opts := minio.PutObjectOptions{ContentType: "application/octet-stream"}

		if sseInfo.Type == "KMS" {
			opts.ServerSideEncryption, _ = encrypt.NewSSEKMS(sseInfo.Key, nil)
		}

		if sseInfo.Type == "SSE" {
			opts.ServerSideEncryption = encrypt.NewSSE()
		}

		if sseInfo.Type == "SSE-C" {
			opts.ServerSideEncryption, err = encrypt.NewSSEC([]byte(sseInfo.Key))
			if err != nil {
				handleHTTPError(w, fmt.Errorf("error setting SSE-C key: %w", err))
				return
			}
		}

		_, err = s3.PutObject(r.Context(), bucketName, path, file, -1, opts)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error putting object: %w", err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
