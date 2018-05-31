package s3manager

import (
	"net/http"

	"github.com/matryer/way"
	"github.com/pkg/errors"
)

// HandleDeleteBucket deletes a bucket.
func HandleDeleteBucket(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")

		err := s3.RemoveBucket(bucketName)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error removing bucket"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
