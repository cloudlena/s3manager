package s3manager

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// GetObjectHandler downloads an object to the client.
func GetObjectHandler(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucketName := vars["bucketName"]
		objectName := vars["objectName"]

		object, err := s3.GetObject(bucketName, objectName, minio.GetObjectOptions{})
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error getting object"))
			return
		}

		w.Header().Set(headerContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", objectName))
		w.Header().Set(HeaderContentType, contentTypeOctetStream)

		_, err = io.Copy(w, object)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error copying object to response writer"))
			return
		}
	})
}
