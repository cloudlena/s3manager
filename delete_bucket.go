package s3manager

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// DeleteBucketHandler deletes a bucket.
func DeleteBucketHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		err := s3.RemoveBucket(bucketName)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error removing bucket"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
