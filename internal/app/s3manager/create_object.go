package s3manager

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"

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
		file, fileHeader, err := r.FormFile("file")
		path := r.FormValue("path")
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting file from form: %w", err))
			return
		}
		defer func() {
			if cErr := file.Close(); cErr != nil {
				log.Printf("error closing file: %v", cErr)
			}
		}()

		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" || contentType == "application/octet-stream" {
			contentType = mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
		}
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		opts := minio.PutObjectOptions{ContentType: contentType}

		if sseInfo.Type == "KMS" {
			opts.ServerSideEncryption, err = encrypt.NewSSEKMS(sseInfo.Key, nil)
			if err != nil {
				handleHTTPError(w, fmt.Errorf("error setting SSE-KMS key: %w", err))
				return
			}
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

		size := fileHeader.Size
		_, err = s3.PutObject(r.Context(), bucketName, path, file, size, opts)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error putting object: %w", err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
