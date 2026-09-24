package s3manager

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// HandleGetBucketPolicy returns the policy of a bucket as JSON, or an empty
// body if the bucket has none.
func HandleGetBucketPolicy(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		policy, err := s3.GetBucketPolicy(r.Context(), bucketName)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting bucket policy: %w", err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err = w.Write([]byte(policy)); err != nil {
			handleHTTPError(w, fmt.Errorf("error writing response: %w", err))
		}
	}
}

// HandlePutBucketPolicy replaces the policy of a bucket with the request body.
func HandlePutBucketPolicy(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		policy, err := io.ReadAll(r.Body)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error reading request body: %w", err))
			return
		}
		err = s3.SetBucketPolicy(r.Context(), bucketName, string(policy))
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error setting bucket policy: %w", err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
