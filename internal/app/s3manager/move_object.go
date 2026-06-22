package s3manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// moveObjectRequest is the body of a move/copy request.
type moveObjectRequest struct {
	// DestInstanceID is the ID of the destination S3 instance. Empty means the
	// currently active instance.
	DestInstanceID string `json:"destInstanceId"`
	// DestBucket is the destination bucket name.
	DestBucket string `json:"destBucket"`
	// DestPath is the destination folder/prefix. The object keeps its original
	// name and is placed under this prefix.
	DestPath string `json:"destPath"`
	// Operation is either "move" (copy then delete the source) or "copy".
	Operation string `json:"operation"`
}

// moveObjectResponse summarizes the result of a move/copy operation.
type moveObjectResponse struct {
	Operation string `json:"operation"`
	Count     int    `json:"count"`
}

// HandleMoveObjectWithManager moves or copies an object (or folder) to another
// bucket/folder, optionally on a different S3 instance.
//
// Within the same instance the copy is performed server-side via CopyObject.
// Across instances the object is streamed through the app (download from the
// source, upload to the destination). When the operation is "move", the source
// objects are removed after a successful copy.
func HandleMoveObjectWithManager(manager *MultiS3Manager, sseInfo SSEType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srcBucket := mux.Vars(r)["bucketName"]
		srcKey := mux.Vars(r)["objectName"]

		var req moveObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleHTTPError(w, fmt.Errorf("error decoding request body: %w", err))
			return
		}

		if req.DestBucket == "" {
			handleHTTPError(w, fmt.Errorf("%w: destination bucket is required", errInvalidMove))
			return
		}

		copyOnly := req.Operation == "copy"

		srcClient := manager.GetCurrentClient()
		destInstanceID := req.DestInstanceID
		if destInstanceID == "" {
			destInstanceID = manager.GetCurrentID()
		}
		sameInstance := destInstanceID == manager.GetCurrentID()

		destClient := srcClient
		if !sameInstance {
			c, err := manager.GetClientByID(destInstanceID)
			if err != nil {
				handleHTTPError(w, fmt.Errorf("%w: %v", errInvalidMove, err))
				return
			}
			destClient = c
		}

		// Normalize the destination prefix so it always ends with a slash (unless
		// it targets the bucket root).
		destPrefix := req.DestPath
		if destPrefix != "" && !strings.HasSuffix(destPrefix, "/") {
			destPrefix += "/"
		}

		// Build the list of (source key -> destination key) pairs. A trailing
		// slash on the source key denotes a folder, which is expanded into all
		// the objects it contains.
		pairs, err := buildMovePairs(r.Context(), srcClient, srcBucket, srcKey, destPrefix)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error listing objects to move: %w", err))
			return
		}

		if len(pairs) == 0 {
			handleHTTPError(w, fmt.Errorf("%w: no objects found at source", errInvalidMove))
			return
		}

		// Guard against a no-op that would delete the source without producing a
		// real copy.
		if !copyOnly && sameInstance && srcBucket == req.DestBucket {
			for _, p := range pairs {
				if p.dest == p.src {
					handleHTTPError(w, fmt.Errorf("%w: source and destination are identical", errInvalidMove))
					return
				}
			}
		}

		// Copy every object first; only delete sources once all copies succeeded
		// so a failure mid-way never loses data.
		for _, p := range pairs {
			if sameInstance {
				_, err = srcClient.CopyObject(r.Context(),
					minio.CopyDestOptions{Bucket: req.DestBucket, Object: p.dest},
					minio.CopySrcOptions{Bucket: srcBucket, Object: p.src})
			} else {
				err = streamCopyObject(r.Context(), srcClient, destClient, srcBucket, p.src, req.DestBucket, p.dest, sseInfo)
			}
			if err != nil {
				handleHTTPError(w, fmt.Errorf("error copying object %q: %w", p.src, err))
				return
			}
		}

		if !copyOnly {
			for _, p := range pairs {
				if rErr := srcClient.RemoveObject(r.Context(), srcBucket, p.src, minio.RemoveObjectOptions{}); rErr != nil {
					handleHTTPError(w, fmt.Errorf("error removing source object %q after copy: %w", p.src, rErr))
					return
				}
			}
		}

		operation := "copy"
		if !copyOnly {
			operation = "move"
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(moveObjectResponse{Operation: operation, Count: len(pairs)}); err != nil {
			log.Printf("error encoding move response: %v", err)
		}
	}
}

// movePair is a single source-to-destination key mapping.
type movePair struct {
	src  string
	dest string
}

// buildMovePairs resolves the source key into one or more source/destination
// key pairs. Files map to a single pair; folders (keys ending in "/") are
// expanded recursively, preserving the folder name and internal structure
// under the destination prefix.
func buildMovePairs(ctx context.Context, s3 S3, bucket, srcKey, destPrefix string) ([]movePair, error) {
	name := path.Base(strings.TrimSuffix(srcKey, "/"))

	if !strings.HasSuffix(srcKey, "/") {
		return []movePair{{src: srcKey, dest: destPrefix + name}}, nil
	}

	// Folder: keep its name under the destination prefix.
	destFolder := destPrefix + name + "/"
	var pairs []movePair
	objectCh := s3.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: srcKey, Recursive: true})
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		rel := strings.TrimPrefix(object.Key, srcKey)
		pairs = append(pairs, movePair{src: object.Key, dest: destFolder + rel})
	}
	return pairs, nil
}

// streamCopyObject copies a single object between two S3 instances by streaming
// it through the app server.
func streamCopyObject(ctx context.Context, src, dest S3, srcBucket, srcKey, destBucket, destKey string, sseInfo SSEType) error {
	object, err := src.GetObject(ctx, srcBucket, srcKey, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("error getting source object: %w", err)
	}
	defer func() {
		if cErr := object.Close(); cErr != nil {
			log.Printf("error closing source object: %v", cErr)
		}
	}()

	info, err := object.Stat()
	if err != nil {
		return fmt.Errorf("error stating source object: %w", err)
	}

	opts := minio.PutObjectOptions{ContentType: info.ContentType}
	if opts.ContentType == "" {
		opts.ContentType = "application/octet-stream"
	}
	if err := applySSE(&opts, sseInfo); err != nil {
		return err
	}

	if _, err := dest.PutObject(ctx, destBucket, destKey, object, info.Size, opts); err != nil {
		return fmt.Errorf("error putting destination object: %w", err)
	}
	return nil
}

// applySSE sets the server-side encryption options on opts according to the
// configured SSE type, mirroring the upload handler.
func applySSE(opts *minio.PutObjectOptions, sseInfo SSEType) error {
	switch sseInfo.Type {
	case "KMS":
		sse, err := encrypt.NewSSEKMS(sseInfo.Key, nil)
		if err != nil {
			return fmt.Errorf("error setting SSE-KMS key: %w", err)
		}
		opts.ServerSideEncryption = sse
	case "SSE":
		opts.ServerSideEncryption = encrypt.NewSSE()
	case "SSE-C":
		sse, err := encrypt.NewSSEC([]byte(sseInfo.Key))
		if err != nil {
			return fmt.Errorf("error setting SSE-C key: %w", err)
		}
		opts.ServerSideEncryption = sse
	}
	return nil
}
