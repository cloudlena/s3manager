package s3manager

import (
	"net/http"

	"github.com/matryer/way"
	"github.com/pkg/errors"
)

// HandleDeleteObject deletes an object.
func HandleDeleteObject(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")
		objectName := way.Param(r.Context(), "objectName")

		err := s3.RemoveObject(bucketName, objectName)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error removing object"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
