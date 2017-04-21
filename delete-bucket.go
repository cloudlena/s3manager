package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// DeleteBucketHandler deletes a bucket
func DeleteBucketHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		err := s3.RemoveBucket(bucketName)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
