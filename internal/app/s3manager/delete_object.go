package s3manager

import (
	"fmt"
	"net/http"

	"github.com/matryer/way"
)

// HandleDeleteObject deletes an object.
func HandleDeleteObject(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")
		objectName := way.Param(r.Context(), "objectName")

		err := s3.RemoveObject(bucketName, objectName)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error removing object: %w", err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
