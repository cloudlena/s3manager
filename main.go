package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mastertinner/s3-manager/adapters"
	"github.com/mastertinner/s3-manager/buckets"
	"github.com/mastertinner/s3-manager/datasources"
	"github.com/mastertinner/s3-manager/objects"
	"github.com/mastertinner/s3-manager/views"
)

func main() {
	s3 := datasources.NewMinioClient()

	logger := log.New(os.Stdout, "request: ", log.Lshortfile)
	router := mux.NewRouter()

	router.
		Methods("GET").
		Path("/").
		Handler(adapters.Adapt(
			views.IndexHandler(),
			adapters.Logging(logger),
		))
	router.
		Methods("GET").
		Path("/buckets").
		Handler(adapters.Adapt(
			views.BucketsHandler(s3),
			adapters.Logging(logger),
		))
	router.
		Methods("GET").
		Path("/buckets/{bucketName}").
		Handler(adapters.Adapt(
			views.BucketHandler(s3),
			adapters.Logging(logger),
		))

	api := router.PathPrefix("/api").Subrouter()

	br := api.PathPrefix("/buckets").Subrouter()
	br.
		Methods("POST").
		Path("").
		Handler(adapters.Adapt(
			buckets.CreateHandler(s3),
			adapters.Logging(logger),
		))
	br.
		Methods("DELETE").
		Path("/{bucketName}").
		Handler(adapters.Adapt(
			buckets.DeleteHandler(s3),
			adapters.Logging(logger),
		))
	br.
		Methods("POST").
		Path("/{bucketName}/objects").
		Handler(adapters.Adapt(
			objects.CreateHandler(s3),
			adapters.Logging(logger),
		))
	br.
		Methods("GET").
		Path("/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			objects.GetHandler(s3),
			adapters.Logging(logger),
		))
	br.
		Methods("DELETE").
		Path("/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			objects.DeleteHandler(s3),
			adapters.Logging(logger),
		))

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, router))
}
