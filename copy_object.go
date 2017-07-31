package s3manager

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	minio "github.com/minio/minio-go"
)

// copyObjectInfo is the information about an object to copy.
type copyObjectInfo struct {
	BucketName       string `json:"bucketName"`
	ObjectName       string `json:"objectName"`
	SourceBucketName string `json:"sourceBucketName"`
	SourceObjectName string `json:"sourceObjectName"`
}

// CopyObjectHandler copies an existing object under a new name.
func CopyObjectHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var copy copyObjectInfo

		err := json.NewDecoder(r.Body).Decode(&copy)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errDecodingBody))
			return
		}

		src := minio.NewSourceInfo(copy.SourceBucketName, copy.SourceObjectName, nil)
		dst, err := minio.NewDestinationInfo(copy.BucketName, copy.ObjectName, nil, nil)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error creating destination for copying"))
			return
		}
		err = s3.CopyObject(dst, src)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error copying object"))
			return
		}

		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(copy)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errEncodingJSON))
			return
		}
	})
}
