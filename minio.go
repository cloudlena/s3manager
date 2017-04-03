package main

import (
	"log"
	"os"

	"github.com/minio/minio-go"
)

// newMinioClient creates a new Minio client
func newMinioClient() *minio.Client {
	var err error
	var client *minio.Client

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "s3.amazonaws.com"
	}

	s3AccessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	if s3AccessKeyID == "" {
		log.Fatal("Please set S3_ACCESS_KEY_ID")
	}

	s3SecretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	if s3SecretAccessKey == "" {
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
