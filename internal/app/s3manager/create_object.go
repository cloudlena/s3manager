package s3manager

import (
	"fmt"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// maxUploadMemory is the amount of an upload kept in memory before it is
// buffered to disk.
const maxUploadMemory = 32 << 20 // 32 MB

// HandleCreateObject uploads a new object.
func HandleCreateObject(s3 S3, sse SSEType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		err := r.ParseMultipartForm(maxUploadMemory)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing multipart form: %w", err))
			return
		}

		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting file from form: %w", err))
			return
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Printf("error closing file: %v", err)
			}
		}()

		opts := minio.PutObjectOptions{ContentType: uploadContentType(fileHeader)}
		opts.ServerSideEncryption, err = serverSideEncryption(sse)
		if err != nil {
			handleHTTPError(w, err)
			return
		}

		_, err = s3.PutObject(r.Context(), bucketName, r.FormValue("path"), file, fileHeader.Size, opts)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error putting object: %w", err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// uploadContentType determines the content type to store an uploaded file under,
// falling back to the file extension if the browser didn't provide a usable one.
func uploadContentType(fileHeader *multipart.FileHeader) string {
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = mime.TypeByExtension(filepath.Ext(fileHeader.Filename))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return contentType
}

// serverSideEncryption builds the encryption to store an object with, or nil if
// server side encryption is not configured.
func serverSideEncryption(sse SSEType) (encrypt.ServerSide, error) {
	switch sse.Type {
	case "KMS":
		encryption, err := encrypt.NewSSEKMS(sse.Key, nil)
		if err != nil {
			return nil, fmt.Errorf("error setting SSE-KMS key: %w", err)
		}
		return encryption, nil
	case "SSE":
		return encrypt.NewSSE(), nil
	case "SSE-C":
		encryption, err := encrypt.NewSSEC([]byte(sse.Key))
		if err != nil {
			return nil, fmt.Errorf("error setting SSE-C key: %w", err)
		}
		return encryption, nil
	default:
		return nil, nil
	}
}
