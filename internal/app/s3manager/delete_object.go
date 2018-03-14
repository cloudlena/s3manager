package s3manager

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// DeleteObjectHandler deletes an object.
func DeleteObjectHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucketName := vars["bucketName"]
		objectName := vars["objectName"]

		err := s3.RemoveObject(bucketName, objectName)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error removing object"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
