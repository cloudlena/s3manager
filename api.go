package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			log.Println("error decoding json:", err)
			return
		}

		err = s.S3.MakeBucket(bucket.Name, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("error making bucket:", err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(bucket)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("error encoding json:", err)
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
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				log.Println("error decoding json:", err)
				return
			}

			var copyConds = minio.NewCopyConditions()
			objectSource := fmt.Sprintf("/%s/%s", copy.SourceBucketName, copy.SourceObjectName)
			fmt.Println(copy)
			fmt.Println(objectSource)
			err = s.S3.CopyObject(copy.BucketName, copy.ObjectName, objectSource, copyConds)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Println("error copying object:", err)
				return
			}

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusCreated)
			err = json.NewEncoder(w).Encode(copy)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Println("error encoding json:", err)
				return
			}
		} else {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				log.Println("error parsing form:", err)
				return
			}

			file, handler, err := r.FormFile("file")
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				log.Println("error getting form file:", err)
				return
			}
			defer file.Close()

			_, err = s.S3.PutObject(vars["bucketName"], handler.Filename, file, "application/octet-stream")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Println("error putting object:", err)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("error removing bucket:", err)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("error removing object:", err)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("error getting object:", err)
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", objectName))
		w.Header().Set("Content-Type", "application/octet-stream")

		_, err = io.Copy(w, object)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Println("error copying object:", err)
			return
		}
	})
}
