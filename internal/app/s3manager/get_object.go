package s3manager

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"io"
	"net/http"
	"path"
)

// HandleGetObject downloads an object to the client.
func HandleGetObject(s3 S3, forceDownload bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]

		object, err := s3.GetObject(r.Context(), bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting object: %w", err))
			return
		}

		fileName := path.Base(objectName)

		if forceDownload {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		_, err = io.Copy(w, object)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error copying object to response writer: %w", err))
			return
		}
	}
}
