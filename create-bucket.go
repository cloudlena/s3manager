package main

import (
	"encoding/json"
	"net/http"

	minio "github.com/minio/minio-go"
)

// CreateBucketHandler creates a new bucket
func CreateBucketHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucket minio.BucketInfo

		err := json.NewDecoder(r.Body).Decode(&bucket)
		if err != nil {
			msg := "error decoding json"
			handleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
			return
		}

		err = s3.MakeBucket(bucket.Name, "")
		if err != nil {
			msg := "error making bucket"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(bucket)
		if err != nil {
			msg := "error encoding json"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}
