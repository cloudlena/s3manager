package s3manager

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// objectMetadata is the JSON shape returned by HandleGetObjectMetadata.
type objectMetadata struct {
	Key          string            `json:"key"`
	VersionID    string            `json:"versionId,omitempty"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"contentType"`
	ETag         string            `json:"etag"`
	LastModified string            `json:"lastModified"`
	StorageClass string            `json:"storageClass,omitempty"`
	IsLatest     bool              `json:"isLatest,omitempty"`
	UserMetadata map[string]string `json:"userMetadata"`
}

// HandleGetObjectMetadata returns metadata for an object (optionally a specific version).
func HandleGetObjectMetadata(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]
		versionID := r.URL.Query().Get("versionId")

		info, err := s3.StatObject(r.Context(), bucketName, objectName, minio.StatObjectOptions{VersionID: versionID})
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting object metadata: %w", err))
			return
		}

		userMetadata := info.UserMetadata
		if len(userMetadata) == 0 {
			userMetadata = userMetadataFromHeaders(info.Metadata)
		}

		writeJSON(w, http.StatusOK, objectMetadata{
			Key:          info.Key,
			VersionID:    info.VersionID,
			Size:         info.Size,
			ContentType:  info.ContentType,
			ETag:         info.ETag,
			LastModified: info.LastModified.Format(time.RFC3339),
			StorageClass: info.StorageClass,
			IsLatest:     info.IsLatest,
			UserMetadata: userMetadata,
		})
	}
}

// userMetadataFromHeaders derives the user metadata from the raw response
// headers, which is where it has to be read from for providers that don't
// populate minio's UserMetadata field (notably AWS S3 itself).
func userMetadataFromHeaders(headers http.Header) map[string]string {
	const prefix = "x-amz-meta-"

	userMetadata := make(map[string]string)
	for key, values := range headers {
		key = strings.ToLower(key)
		if !strings.HasPrefix(key, prefix) || len(values) == 0 {
			continue
		}
		userMetadata[strings.TrimPrefix(key, prefix)] = values[0]
	}

	return userMetadata
}
