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
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	s := &Server{
		S3: NewMinioClient(),
	}

	router := mux.NewRouter()

	router.
		Methods("GET").
		Path("/").
		Handler(Adapt(IndexHandler(), Logger()))
	router.
		Methods("GET").
		Path("/buckets").
		Handler(Adapt(s.BucketsPageHandler(), Logger()))
	router.
		Methods("GET").
		Path("/buckets/{bucketName}").
		Handler(Adapt(s.BucketPageHandler(), Logger()))

	api := router.PathPrefix("/api").Subrouter()

	buckets := api.PathPrefix("/buckets").Subrouter()
	buckets.
		Methods("POST").
		Path("").
		Handler(Adapt(s.CreateBucketHandler(), Logger()))
	buckets.
		Methods("DELETE").
		Path("/{bucketName}").
		Handler(Adapt(s.DeleteBucketHandler(), Logger()))
	buckets.
		Methods("POST").
		Path("/{bucketName}/objects").
		Handler(Adapt(s.CreateObjectHandler(), Logger()))
	buckets.
		Methods("GET").
		Path("/{bucketName}/objects/{objectName}").
		Handler(Adapt(s.GetObjectHandler(), Logger()))
	buckets.
		Methods("DELETE").
		Path("/{bucketName}/objects/{objectName}").
		Handler(Adapt(s.DeleteObjectHandler(), Logger()))

	log.Fatal(http.ListenAndServe(":"+port, router))
}
