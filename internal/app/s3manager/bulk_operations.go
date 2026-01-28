package s3manager

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// BulkDeleteRequest represents the request body for bulk delete
type BulkDeleteRequest struct {
	Keys []string `json:"keys"`
}

// BulkDownloadRequest represents the request body for bulk download
type BulkDownloadRequest struct {
	Keys []string `json:"keys"`
}

// HandleBulkDeleteObjects deletes multiple objects from a bucket.
func HandleBulkDeleteObjects(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		var req BulkDeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing request: %w", err))
			return
		}

		if len(req.Keys) == 0 {
			http.Error(w, "no keys provided", http.StatusBadRequest)
			return
		}

		// Create a channel for objects to delete
		objectsCh := make(chan minio.ObjectInfo)

		// Send object names to the channel
		go func() {
			defer close(objectsCh)
			for _, key := range req.Keys {
				objectsCh <- minio.ObjectInfo{Key: key}
			}
		}()

		// Remove objects
		errorCh := s3.RemoveObjects(r.Context(), bucketName, objectsCh, minio.RemoveObjectsOptions{})

		// Check for errors
		for err := range errorCh {
			if err.Err != nil {
				handleHTTPError(w, fmt.Errorf("error removing object %s: %w", err.ObjectName, err.Err))
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"success": true}`)); err != nil {
			// Response already sent, can only log the error
			fmt.Printf("error writing response: %v\n", err)
		}
	}
}

// HandleBulkDownloadObjects downloads multiple objects as a ZIP archive.
func HandleBulkDownloadObjects(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		// Parse the form to get the keys
		if err := r.ParseForm(); err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing form: %w", err))
			return
		}

		keysJSON := r.FormValue("keys")
		var keys []string
		if err := json.Unmarshal([]byte(keysJSON), &keys); err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing keys: %w", err))
			return
		}

		if len(keys) == 0 {
			http.Error(w, "no keys provided", http.StatusBadRequest)
			return
		}

		// Set headers for ZIP download
		timestamp := time.Now().Format("20060102-150405")
		zipFilename := fmt.Sprintf("%s-%s.zip", bucketName, timestamp)
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFilename))

		// Create a new ZIP writer
		zipWriter := zip.NewWriter(w)
		defer func() {
			if err := zipWriter.Close(); err != nil {
				// Can't return HTTP error at this point, just log
				fmt.Printf("error closing zip writer: %v\n", err)
			}
		}()

		// Add each object to the ZIP
		for _, key := range keys {
			// Get the object from S3
			object, err := s3.GetObject(r.Context(), bucketName, key, minio.GetObjectOptions{})
			if err != nil {
				// Log error but continue with other files
				continue
			}

			// Get object info to check if it's valid
			_, err = object.Stat()
			if err != nil {
				_ = object.Close()
				continue
			}

			// Create a file in the ZIP
			zipFile, err := zipWriter.Create(key)
			if err != nil {
				_ = object.Close()
				continue
			}

			// Copy the object content to the ZIP file
			_, err = io.Copy(zipFile, object)
			_ = object.Close()
			if err != nil {
				// Error writing to ZIP, but we can't return HTTP error at this point
				continue
			}
		}
	}
}
