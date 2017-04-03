package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// DeleteBucketHandler deletes a bucket
func DeleteBucketHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := s3.RemoveBucket(vars["bucketName"])
		if err != nil {
			msg := "error removing bucket"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
