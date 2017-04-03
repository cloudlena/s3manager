package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mastertinner/adapters"
	"github.com/mastertinner/adapters/logging"
)

func main() {
	s3 := NewMinioClient()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	router := mux.NewRouter()

	router.
		Methods("GET").
		Path("/").
		Handler(adapters.Adapt(
			IndexViewHandler(),
			logging.Handler(logger),
		))
	router.
		Methods("GET").
		Path("/buckets").
		Handler(adapters.Adapt(
			BucketsViewHandler(s3),
			logging.Handler(logger),
		))
	router.
		Methods("GET").
		Path("/buckets/{bucketName}").
		Handler(adapters.Adapt(
			BucketViewHandler(s3),
			logging.Handler(logger),
		))

	api := router.PathPrefix("/api").Subrouter()

	br := api.PathPrefix("/buckets").Subrouter()
	br.
		Methods("POST").
		Path("").
		Handler(adapters.Adapt(
			CreateBucketHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods("DELETE").
		Path("/{bucketName}").
		Handler(adapters.Adapt(
			DeleteBucketHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods("POST").
		Path("/{bucketName}/objects").
		Handler(adapters.Adapt(
			CreateObjectHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods("GET").
		Path("/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			GetObjectHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods("DELETE").
		Path("/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			DeleteObjectHandler(s3),
			logging.Handler(logger),
		))

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, router))
}
