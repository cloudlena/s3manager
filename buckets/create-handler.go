package buckets

import (
	"encoding/json"
	"net/http"

	"github.com/mastertinner/s3-manager/web"
	minio "github.com/minio/minio-go"
)

// CreateHandler creates a new bucket
func CreateHandler(s3 *minio.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucket minio.BucketInfo

		err := json.NewDecoder(r.Body).Decode(&bucket)
		if err != nil {
			msg := "error decoding json"
			web.HandleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
			return
		}

		err = s3.MakeBucket(bucket.Name, "")
		if err != nil {
			msg := "error making bucket"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(bucket)
		if err != nil {
			msg := "error encoding json"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}
