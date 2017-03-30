package objects

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mastertinner/s3-manager/web"
	minio "github.com/minio/minio-go"
)

// CopyObjectInfo is the information about an object to copy
type CopyObjectInfo struct {
	BucketName       string `json:"bucketName"`
	ObjectName       string `json:"objectName"`
	SourceBucketName string `json:"sourceBucketName"`
	SourceObjectName string `json:"sourceObjectName"`
}

// CreateHandler allows to upload a new object
func CreateHandler(s3 *minio.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if r.Header.Get("Content-Type") == "application/json" {
			var copy CopyObjectInfo

			err := json.NewDecoder(r.Body).Decode(&copy)
			if err != nil {
				msg := "error decoding json"
				web.HandleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
				return
			}

			var copyConds = minio.NewCopyConditions()
			objectSource := fmt.Sprintf("/%s/%s", copy.SourceBucketName, copy.SourceObjectName)
			err = s3.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
			if err != nil {
				msg := "error copying object"
				web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusCreated)

			err = json.NewEncoder(w).Encode(copy)
			if err != nil {
				msg := "error encoding json"
				web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}
		} else {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				msg := "error parsing form"
				web.HandleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
				return
			}

			file, handler, err := r.FormFile("file")
			if err != nil {
				msg := "error getting form file"
				web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}
			defer file.Close()

			_, err = s3.PutObject(vars["bucketName"], handler.Filename, file, "application/octet-stream")
			if err != nil {
				msg := "error putting object"
				web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		}
	})
}
