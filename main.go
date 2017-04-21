package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mastertinner/adapters"
	"github.com/mastertinner/adapters/logging"
)

const (
	tmplDirectory            = "templates"
	headerContentType        = "Content-Type"
	headerContentDisposition = "Content-Disposition"
	contentTypeJSON          = "application/json"
	contentTypeMultipartForm = "multipart/form-data"
	contentTypeOctetStream   = "application/octet-stream"
)

func main() {
	s3, err := newMinioClient()
	if err != nil {
		log.Fatalln("error creating s3 client:", err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	r := mux.NewRouter()
	r.
		Methods(http.MethodGet).
		Path("/").
		Handler(adapters.Adapt(
			http.RedirectHandler("/buckets", http.StatusPermanentRedirect),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodGet).
		Path("/buckets").
		Handler(adapters.Adapt(
			BucketsViewHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodGet).
		Path("/buckets/{bucketName}").
		Handler(adapters.Adapt(
			BucketViewHandler(s3),
			logging.Handler(logger),
		))

	api := r.PathPrefix("/api").Subrouter()

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
		Headers(headerContentType, contentTypeJSON).
		Path("/{bucketName}/objects").
		Handler(adapters.Adapt(
			CopyObjectHandler(s3),
			logging.Handler(logger),
		))
	br.
		Methods(http.MethodPost).
		HeadersRegexp(headerContentType, contentTypeMultipartForm).
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
	log.Fatal(http.ListenAndServe(":"+port, r))
}
