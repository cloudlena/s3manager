package main

import (
	"log"
	"os"

	"github.com/minio/minio-go"
)

// NewMinioClient creates a new Minio client
func NewMinioClient() *minio.Client {
	var err error
	var client *minio.Client

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if len(s3Endpoint) == 0 {
		s3Endpoint = "s3.amazonaws.com"
	}

	s3AccessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	if len(s3AccessKeyID) == 0 {
		log.Fatal("Please set S3_ACCESS_KEY_ID")
	}

	s3SecretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	if len(s3SecretAccessKey) == 0 {
		log.Fatal("Please set S3_SECRET_ACCESS_KEY")
	}

	if os.Getenv("V2_SIGNING") == "true" {
		client, err = minio.NewV2(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	} else {
		client, err = minio.New(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	}
	if err != nil {
		log.Fatal(err)
	}

	return client
}
