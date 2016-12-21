package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
)

// createBucketHandler creates a new bucket
func createBucketHandler(w http.ResponseWriter, r *http.Request) {
	var bucket minio.BucketInfo
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err = r.Body.Close(); err != nil {
		panic(err)
	}
	err = json.Unmarshal(body, &bucket)
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
		panic(err)
	}
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

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		panic(err)
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	defer file.Close()

	_, err = minioClient.PutObject(vars["bucketName"], handler.Filename, file, "application/octet-stream")
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusCreated)
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
