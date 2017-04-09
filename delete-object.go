package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// DeleteObjectHandler deletes an object
func DeleteObjectHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := s3.RemoveObject(vars["bucketName"], vars["objectName"])
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}
