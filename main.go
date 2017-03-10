package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
)

// Server is a server containing a minio client
type Server struct {
	s3 *minio.Client
}

func main() {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	s := &Server{
		s3: NewMinioClient(),
	}

	router := mux.NewRouter().StrictSlash(true)

	router.
		Methods("GET").
		Path("/").
		HandlerFunc(Chain(IndexHandler, Logger()))
	router.
		Methods("GET").
		Path("/buckets").
		HandlerFunc(Chain(s.BucketsPageHandler, Logger()))
	router.
		Methods("GET").
		Path("/buckets/{bucketName}").
		HandlerFunc(Chain(s.BucketPageHandler, Logger()))

	api := router.PathPrefix("/api").Subrouter()

	buckets := api.PathPrefix("/buckets").Subrouter()
	buckets.
		Methods("POST").
		Path("").
		HandlerFunc(Chain(s.CreateBucketHandler, Logger()))
	buckets.
		Methods("DELETE").
		Path("/{bucketName}").
		HandlerFunc(Chain(s.DeleteBucketHandler, Logger()))
	buckets.
		Methods("POST").
		Path("/{bucketName}/objects").
		HandlerFunc(Chain(s.CreateObjectHandler, Logger()))
	buckets.
		Methods("GET").
		Path("/{bucketName}/objects/{objectName}").
		HandlerFunc(Chain(s.GetObjectHandler, Logger()))
	buckets.
		Methods("DELETE").
		Path("/{bucketName}/objects/{objectName}").
		HandlerFunc(Chain(s.DeleteObjectHandler, Logger()))

	log.Fatal(http.ListenAndServe(":"+port, router))
}
