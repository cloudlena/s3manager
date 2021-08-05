package s3manager

import (
	"fmt"
	"net/http"

	"github.com/matryer/way"
)

// HandleDeleteBucket deletes a bucket.
func HandleDeleteBucket(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")

		err := s3.RemoveBucket(r.Context(), bucketName)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error removing bucket: %w", err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
