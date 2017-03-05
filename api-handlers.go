package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
)

// CopyObjectInfo is the information about an object to copy
type CopyObjectInfo struct {
	BucketName       string `json:"bucketName"`
	ObjectName       string `json:"objectName"`
	SourceBucketName string `json:"sourceBucketName"`
	SourceObjectName string `json:"sourceObjectName"`
}

// createBucketHandler creates a new bucket
func createBucketHandler(w http.ResponseWriter, r *http.Request) {
	var bucket minio.BucketInfo

	err := json.NewDecoder(r.Body).Decode(&bucket)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	err = minioClient.MakeBucket(bucket.Name, "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(bucket)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// deleteBucketHandler deletes a bucket
func deleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := minioClient.RemoveBucket(vars["bucketName"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// getObjectHandler downloads an object to the client
func getObjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectName := vars["objectName"]

	object, err := minioClient.GetObject(vars["bucketName"], objectName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", objectName))
	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = io.Copy(w, object)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// createObjectHandler allows to upload a new object
func createObjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if r.Header.Get("Content-Type") == "application/json" {
		var copy CopyObjectInfo

		err := json.NewDecoder(r.Body).Decode(&copy)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		var copyConds = minio.NewCopyConditions()
		objectSource := fmt.Sprintf("/%s/%s", copy.SourceBucketName, copy.SourceObjectName)
		fmt.Println(copy)
		fmt.Println(objectSource)
		err = minioClient.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(copy)
		if err != nil {
			panic(err)
		}
	} else {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		defer file.Close()

		_, err = minioClient.PutObject(vars["bucketName"], handler.Filename, file, "application/octet-stream")
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// deleteObjectHandler deletes an object
func deleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := minioClient.RemoveObject(vars["bucketName"], vars["objectName"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
