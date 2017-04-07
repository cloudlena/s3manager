package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

// GetObjectHandler downloads an object to the client
func GetObjectHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		objectName := vars["objectName"]

		object, err := s3.GetObject(vars["bucketName"], objectName)
		if err != nil {
			msg := "error getting object"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set(headerContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", objectName))
		w.Header().Set(headerContentType, contentTypeOctetStream)

		_, err = io.Copy(w, object)
		if err != nil {
			msg := "error copying object"
			code := http.StatusInternalServerError
			if err.Error() == "The specified key does not exist." {
				msg = "object not found"
				code = http.StatusNotFound
			}
			if err.Error() == "The specified bucket does not exist." {
				msg = "bucket not found"
				code = http.StatusNotFound
			}

			handleHTTPError(w, msg, err, code)
			return
		}
	})
}
