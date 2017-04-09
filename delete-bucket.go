package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// DeleteBucketHandler deletes a bucket
func DeleteBucketHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := s3.RemoveBucket(vars["bucketName"])
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		code := http.StatusNoContent
		w.WriteHeader(code)
		fmt.Fprint(w, http.StatusText(code))
	})
}
