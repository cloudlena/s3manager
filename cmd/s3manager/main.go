package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mastertinner/adapters/logging"
	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
)

func main() {
	accessKeyID, ok := os.LookupEnv("ACCESS_KEY_ID")
	if !ok {
		log.Fatal("please provide ACCESS_KEY_ID")
	}

	secretAccessKey, ok := os.LookupEnv("SECRET_ACCESS_KEY")
	if !ok {
		log.Fatal("please provide SECRET_ACCESS_KEY")
	}

	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	endpoint, ok := os.LookupEnv("ENDPOINT")
	if !ok {
		endpoint = "s3.amazonaws.com"
	}

	tmplDir := filepath.Join("web", "template")

	// Set up S3 client
	s3, err := minio.New(endpoint, accessKeyID, secretAccessKey, true)
	if err != nil {
		log.Fatalln(fmt.Errorf("error creating s3 client: %w", err))
	}

	// Set up router
	r := way.NewRouter()
	r.Handle(http.MethodGet, "/", http.RedirectHandler("/buckets", http.StatusPermanentRedirect))
	r.Handle(http.MethodGet, "/buckets", s3manager.HandleBucketsView(s3, tmplDir))
	r.Handle(http.MethodGet, "/buckets/:bucketName", s3manager.HandleBucketView(s3, tmplDir))
	r.Handle(http.MethodPost, "/api/buckets", s3manager.HandleCreateBucket(s3))
	r.Handle(http.MethodDelete, "/api/buckets/:bucketName", s3manager.HandleDeleteBucket(s3))
	r.Handle(http.MethodPost, "/api/buckets/:bucketName/objects", s3manager.HandleCreateObject(s3))
	r.Handle(http.MethodGet, "/api/buckets/:bucketName/objects/:objectName", s3manager.HandleGetObject(s3))
	r.Handle(http.MethodDelete, "/api/buckets/:bucketName/objects/:objectName", s3manager.HandleDeleteObject(s3))

	lr := logging.Handler(os.Stdout)(r)
	log.Fatal(http.ListenAndServe(":"+port, lr))
}
