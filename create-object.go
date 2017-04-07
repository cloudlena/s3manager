package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
)

// CopyObjectInfo is the information about an object to copy
type CopyObjectInfo struct {
	BucketName       string `json:"bucketName"`
	ObjectName       string `json:"objectName"`
	SourceBucketName string `json:"sourceBucketName"`
	SourceObjectName string `json:"sourceObjectName"`
}

// CreateObjectHandler allows to upload a new object
func CreateObjectHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if strings.Contains(r.Header.Get(headerContentType), contentTypeJSON) {
			var copy CopyObjectInfo

			err := json.NewDecoder(r.Body).Decode(&copy)
			if err != nil {
				msg := "error decoding json"
				handleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
				return
			}

			copyConds := minio.NewCopyConditions()
			objectSource := fmt.Sprintf("/%s/%s", copy.SourceBucketName, copy.SourceObjectName)
			err = s3.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
			if err != nil {
				msg := "error copying object"
				handleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}

			w.Header().Set(headerContentType, contentTypeJSON)
			w.WriteHeader(http.StatusCreated)

			err = json.NewEncoder(w).Encode(copy)
			if err != nil {
				msg := "error encoding json"
				handleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}
		} else {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				msg := "error parsing form"
				handleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
				return
			}

			file, handler, err := r.FormFile("file")
			if err != nil {
				msg := "error getting form file"
				handleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}
			defer file.Close()

			_, err = s3.PutObject(vars["bucketName"], handler.Filename, file, contentTypeOctetStream)
			if err != nil {
				msg := "error putting object"
				handleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		}
	})
}
