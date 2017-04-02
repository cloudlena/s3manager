package objects

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mastertinner/s3-manager/datasources"
	"github.com/mastertinner/s3-manager/utils"
)

// DeleteHandler deletes an object
func DeleteHandler(s3 datasources.S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := s3.RemoveObject(vars["bucketName"], vars["objectName"])
		if err != nil {
			msg := "error removing object"
			utils.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
