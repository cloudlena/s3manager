package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	minio "github.com/minio/minio-go"
)

// CopyObjectInfo is the information about an object to copy
type CopyObjectInfo struct {
	BucketName       string `json:"bucketName"`
	ObjectName       string `json:"objectName"`
	SourceBucketName string `json:"sourceBucketName"`
	SourceObjectName string `json:"sourceObjectName"`
}

// CopyObjectHandler allows to copy an existing object
func CopyObjectHandler(s3 S3Client) http.Handler {
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
