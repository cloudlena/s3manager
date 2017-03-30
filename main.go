package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go"
)

// Server is a server containing a minio client
type Server struct {
	S3 *minio.Client
}

func main() {
	s := &Server{
		S3: NewMinioClient(),
	}

	logger := log.New(os.Stdout, "request: ", log.Lshortfile)
	router := mux.NewRouter()

	router.
		Methods("GET").
		Path("/").
		Handler(Adapt(IndexHandler(), Logging(logger)))
	router.
		Methods("GET").
		Path("/buckets").
		Handler(Adapt(s.BucketsPageHandler(), Logging(logger)))
	router.
		Methods("GET").
		Path("/buckets/{bucketName}").
		Handler(Adapt(s.BucketPageHandler(), Logging(logger)))

	api := router.PathPrefix("/api").Subrouter()

	buckets := api.PathPrefix("/buckets").Subrouter()
	buckets.
		Methods("POST").
		Path("").
		Handler(Adapt(s.CreateBucketHandler(), Logging(logger)))
	buckets.
		Methods("DELETE").
		Path("/{bucketName}").
		Handler(Adapt(s.DeleteBucketHandler(), Logging(logger)))
	buckets.
		Methods("POST").
		Path("/{bucketName}/objects").
		Handler(Adapt(s.CreateObjectHandler(), Logging(logger)))
	buckets.
		Methods("GET").
		Path("/{bucketName}/objects/{objectName}").
		Handler(Adapt(s.GetObjectHandler(), Logging(logger)))
	buckets.
		Methods("DELETE").
		Path("/{bucketName}/objects/{objectName}").
		Handler(Adapt(s.DeleteObjectHandler(), Logging(logger)))

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, router))
}
