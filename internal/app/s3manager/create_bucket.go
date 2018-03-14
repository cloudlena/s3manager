package s3manager

import (
	"encoding/json"
	"net/http"

	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// CreateBucketHandler creates a new bucket.
func CreateBucketHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucket minio.BucketInfo
		err := json.NewDecoder(r.Body).Decode(&bucket)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errDecodingBody))
			return
		}

		err = s3.MakeBucket(bucket.Name, "")
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error making bucket"))
			return
		}

		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(bucket)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errEncodingJSON))
			return
		}
	})
}
