package s3manager

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	minio "github.com/minio/minio-go"
)

// CopyObjectHandler copies an existing object under a new name.
func CopyObjectHandler(s3 S3) http.Handler {
	// request is the information about an object to copy.
	type request struct {
		BucketName       string `json:"bucketName"`
		ObjectName       string `json:"objectName"`
		SourceBucketName string `json:"sourceBucketName"`
		SourceObjectName string `json:"sourceObjectName"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req request
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error decoding body JSON"))
			return
		}

		src := minio.NewSourceInfo(req.SourceBucketName, req.SourceObjectName, nil)
		dst, err := minio.NewDestinationInfo(req.BucketName, req.ObjectName, nil, nil)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error creating destination for copying"))
			return
		}
		err = s3.CopyObject(dst, src)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error copying object"))
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(req)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error encoding JSON"))
			return
		}
	})
}
