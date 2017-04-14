package main

import (
	"encoding/json"
	"fmt"
	"net/http"

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

// CreateObjectFromJSONHandler allows to copy an existing object
func CreateObjectFromJSONHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var copy CopyObjectInfo

		err := json.NewDecoder(r.Body).Decode(&copy)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		copyConds := minio.NewCopyConditions()
		objectSource := fmt.Sprintf("/%s/%s", copy.SourceBucketName, copy.SourceObjectName)
		err = s3.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.Header().Set(headerContentType, contentTypeJSON)
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(copy)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}
	})
}

// CreateObjectFromFormHandler allows to upload a new object
func CreateObjectFromFormHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			handleHTTPError(w, http.StatusUnprocessableEntity, err)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer file.Close()

		_, err = s3.PutObject(vars["bucketName"], handler.Filename, file, contentTypeOctetStream)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})
}
