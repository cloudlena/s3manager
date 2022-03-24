package s3manager

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// HandleCreateObject uploads a new object.
func HandleCreateObject(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

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
		defer func(file multipart.File) {
			if err = file.Close(); err != nil {
				log.Fatal(fmt.Errorf("file cannot be closed: %w", err))
			}
		}(file)

		opts := minio.PutObjectOptions{ContentType: "application/octet-stream"}
		_, err = s3.PutObject(r.Context(), bucketName, header.Filename, file, -1, opts)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error putting object: %w", err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
