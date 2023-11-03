package s3manager

import (
	"context"
	"encoding/json"
	"net/http"
)

func HandleListBuckets(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		bucketInfo, err := s3.ListBuckets(ctx)
		if err != nil {
			handleHTTPError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(bucketInfo)
		if err != nil {
			handleHTTPError(w, err)
			return
		}
	}
}
