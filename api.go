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

// CreateBucketHandler creates a new bucket
func (s *Server) CreateBucketHandler(w http.ResponseWriter, r *http.Request) {
	var bucket minio.BucketInfo

	err := json.NewDecoder(r.Body).Decode(&bucket)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	err = s.s3.MakeBucket(bucket.Name, "")
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

// DeleteBucketHandler deletes a bucket
func (s *Server) DeleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := s.s3.RemoveBucket(vars["bucketName"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetObjectHandler downloads an object to the client
func (s *Server) GetObjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectName := vars["objectName"]

	object, err := s.s3.GetObject(vars["bucketName"], objectName)
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

// CreateObjectHandler allows to upload a new object
func (s *Server) CreateObjectHandler(w http.ResponseWriter, r *http.Request) {
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
		err = s.s3.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
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

		_, err = s.s3.PutObject(vars["bucketName"], handler.Filename, file, "application/octet-stream")
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

// DeleteObjectHandler deletes an object
func (s *Server) DeleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	err := s.s3.RemoveObject(vars["bucketName"], vars["objectName"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
