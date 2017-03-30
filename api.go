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
func (s *Server) CreateBucketHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var bucket minio.BucketInfo

		err := json.NewDecoder(r.Body).Decode(&bucket)
		if err != nil {
			msg := "error decoding json"
			handleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
			return
		}

		err = s.S3.MakeBucket(bucket.Name, "")
		if err != nil {
			msg := "error making bucket"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(bucket)
		if err != nil {
			msg := "error encoding json"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}

// CreateObjectHandler allows to upload a new object
func (s *Server) CreateObjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if r.Header.Get("Content-Type") == "application/json" {
			var copy CopyObjectInfo

			err := json.NewDecoder(r.Body).Decode(&copy)
			if err != nil {
				msg := "error decoding json"
				handleHTTPError(w, msg, err, http.StatusUnprocessableEntity)
				return
			}

			var copyConds = minio.NewCopyConditions()
			objectSource := fmt.Sprintf("/%s/%s", copy.SourceBucketName, copy.SourceObjectName)
			err = s.S3.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
			if err != nil {
				msg := "error copying object"
				handleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
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

			_, err = s.S3.PutObject(vars["bucketName"], handler.Filename, file, "application/octet-stream")
			if err != nil {
				msg := "error putting object"
				handleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		}
	})
}

// DeleteBucketHandler deletes a bucket
func (s *Server) DeleteBucketHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := s.S3.RemoveBucket(vars["bucketName"])
		if err != nil {
			msg := "error removing bucket"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

// DeleteObjectHandler deletes an object
func (s *Server) DeleteObjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		err := s.S3.RemoveObject(vars["bucketName"], vars["objectName"])
		if err != nil {
			msg := "error removing object"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

// GetObjectHandler downloads an object to the client
func (s *Server) GetObjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		objectName := vars["objectName"]

		object, err := s.S3.GetObject(vars["bucketName"], objectName)
		if err != nil {
			msg := "error getting object"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", objectName))
		w.Header().Set("Content-Type", "application/octet-stream")

		_, err = io.Copy(w, object)
		if err != nil {
			msg := "error copying object"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}
