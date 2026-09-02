package s3manager

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// HandleBulkDeleteObjects deletes multiple objects from a bucket.
func HandleBulkDeleteObjects(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		var req struct {
			Keys []string `json:"keys"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing request: %w", err))
			return
		}

		if len(req.Keys) == 0 {
			http.Error(w, "no keys provided", http.StatusBadRequest)
			return
		}

		objectsCh := make(chan minio.ObjectInfo)
		go func() {
			defer close(objectsCh)
			for _, key := range req.Keys {
				objectsCh <- minio.ObjectInfo{Key: key}
			}
		}()

		for err := range s3.RemoveObjects(r.Context(), bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
			if err.Err != nil {
				handleHTTPError(w, fmt.Errorf("error removing object %s: %w", err.ObjectName, err.Err))
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

// HandleBulkDownloadObjects downloads multiple objects as a ZIP archive.
// Objects that cannot be read are skipped: the archive is already being
// streamed to the client, so there is no way to report an error anymore.
func HandleBulkDownloadObjects(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		if err := r.ParseForm(); err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing form: %w", err))
			return
		}

		var keys []string
		if err := json.Unmarshal([]byte(r.FormValue("keys")), &keys); err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing keys: %w", err))
			return
		}

		if len(keys) == 0 {
			http.Error(w, "no keys provided", http.StatusBadRequest)
			return
		}

		zipName := fmt.Sprintf("%s-%s.zip", bucketName, time.Now().Format("20060102-150405"))
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipName))

		zipWriter := zip.NewWriter(w)
		defer func() {
			if err := zipWriter.Close(); err != nil {
				log.Printf("error closing zip writer: %v", err)
			}
		}()

		for _, key := range keys {
			if err := addObjectToZip(r.Context(), s3, zipWriter, bucketName, key); err != nil {
				log.Printf("error adding object %s to zip: %v", key, err)
			}
		}
	}
}

// addObjectToZip streams a single object into the ZIP archive.
func addObjectToZip(ctx context.Context, s3 S3, zipWriter *zip.Writer, bucketName, key string) error {
	object, err := s3.GetObject(ctx, bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("error getting object: %w", err)
	}
	defer func() { _ = object.Close() }()

	if _, err := object.Stat(); err != nil {
		return fmt.Errorf("error getting object info: %w", err)
	}

	zipFile, err := zipWriter.Create(key)
	if err != nil {
		return fmt.Errorf("error creating zip entry: %w", err)
	}

	if _, err := io.Copy(zipFile, object); err != nil {
		return fmt.Errorf("error writing zip entry: %w", err)
	}

	return nil
}
