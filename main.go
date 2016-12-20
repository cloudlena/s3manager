package main

import (
	"log"
	"net/http"
	"os"

	"github.com/minio/minio-go"
)

var minioClient *minio.Client

func main() {
	var err error

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}
	s3AccessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	if len(s3AccessKeyID) == 0 {
		log.Fatalln("Please set S3_ACCESS_KEY_ID")
	}

	s3SecretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	if len(s3SecretAccessKey) == 0 {
		log.Fatalln("Please set S3_SECRET_ACCESS_KEY")
	}

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if len(s3Endpoint) == 0 {
		s3Endpoint = "s3.amazonaws.com"
	}

	if os.Getenv("V2_SIGNING") == "true" {
		minioClient, err = minio.NewV2(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	} else {
		minioClient, err = minio.New(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	}
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/buckets", bucketsHandler)
	http.HandleFunc("/buckets/", bucketHandler)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
