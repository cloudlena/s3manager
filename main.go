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
	s3 := newMinioClient()
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	router := mux.NewRouter()

	router.
		Methods(http.MethodGet).
		Path("/").
		Handler(adapters.Adapt(
			IndexViewHandler(),
			logging.Handler(logger),
		))
	router.
		Methods(http.MethodGet).
		Path("/buckets").
		Handler(adapters.Adapt(
			BucketsViewHandler(s3),
			logging.Handler(logger),
		))
	router.
		Methods(http.MethodGet).
		Path("/buckets/{bucketName}").
		Handler(adapters.Adapt(
			BucketViewHandler(s3),
			logging.Handler(logger),
		))

	api := router.PathPrefix("/api").Subrouter()

	br := api.PathPrefix("/buckets").Subrouter()
	br.
		Methods(http.MethodPost).
		Path("").
		Handler(adapters.Adapt(
			CreateBucketHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods(http.MethodDelete).
		Path("/{bucketName}").
		Handler(adapters.Adapt(
			DeleteBucketHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods(http.MethodPost).
		Path("/{bucketName}/objects").
		Handler(adapters.Adapt(
			CreateObjectHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods(http.MethodGet).
		Path("/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			GetObjectHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods(http.MethodDelete).
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
