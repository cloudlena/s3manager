package s3manager

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

		// Cancelling on return releases the goroutine feeding the removal
		// if an object fails to be removed before everything was fed to it.
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		objectsCh := make(chan minio.ObjectInfo)
		listErrCh := make(chan error, 1)
		go func() {
			defer close(objectsCh)
			listErrCh <- forEachKey(ctx, s3, bucketName, req.Keys, func(key string) error {
				select {
				case objectsCh <- minio.ObjectInfo{Key: key}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
		}()

		for err := range s3.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
			if err.Err != nil {
				handleHTTPError(w, fmt.Errorf("error removing object %s: %w", err.ObjectName, err.Err))
				return
			}
		}

		if err := <-listErrCh; err != nil {
			handleHTTPError(w, err)
			return
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

		err := forEachKey(r.Context(), s3, bucketName, keys, func(key string) error {
			if err := addObjectToZip(r.Context(), s3, zipWriter, bucketName, key); err != nil {
				log.Printf("error adding object %s to zip: %v", key, err)
			}
			return nil
		})
		if err != nil {
			log.Printf("error listing objects for zip: %v", err)
		}
	}
}

// forEachKey calls fn with every object key that the given keys stand for. A
// key ending in "/" is a folder, which stands for every object below it, the
// folder marker included; any other key stands for just itself. It stops at
// the first error that fn or listing a folder returns.
func forEachKey(ctx context.Context, s3 S3, bucketName string, keys []string, fn func(key string) error) error {
	for _, key := range keys {
		if !strings.HasSuffix(key, "/") {
			if err := fn(key); err != nil {
				return err
			}
			continue
		}

		if err := forEachObjectInFolder(ctx, s3, bucketName, key, fn); err != nil {
			return fmt.Errorf("error listing folder %s: %w", key, err)
		}
	}

	return nil
}

// forEachObjectInFolder calls fn with the key of every object below prefix.
// It falls back to a ListObjects V1 request on an empty result for the same
// reason as listWithV1Fallback: otherwise a folder on a provider that answers
// V2 listings with nothing would silently come out as empty.
func forEachObjectInFolder(ctx context.Context, s3 S3, bucketName, prefix string, fn func(key string) error) error {
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	found, err := forEachListedObject(ctx, s3, bucketName, opts, fn)
	if err != nil || found {
		return err
	}

	opts.UseV1 = true
	found, err = forEachListedObject(ctx, s3, bucketName, opts, fn)
	if !found {
		// A provider that rejects V1 is one that meant its empty answer.
		return nil
	}

	return err
}

// forEachListedObject calls fn with the key of every object a listing returns,
// and reports whether it returned any. The context handed to ListObjects is
// cancelled on return, which releases minio's producer goroutine when an error
// cuts the listing short.
func forEachListedObject(ctx context.Context, s3 S3, bucketName string, opts minio.ListObjectsOptions, fn func(key string) error) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	found := false
	for object := range s3.ListObjects(ctx, bucketName, opts) {
		if object.Err != nil {
			return found, object.Err
		}
		found = true
		if err := fn(object.Key); err != nil {
			return found, err
		}
	}

	return found, nil
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
