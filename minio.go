package main

import (
	"errors"
	"os"

	"github.com/minio/minio-go"
)

// newMinioClient creates a new Minio client
func newMinioClient() (*minio.Client, error) {
	var err error
	var c *minio.Client

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = "s3.amazonaws.com"
	}

	s3AccessKeyID := os.Getenv("S3_ACCESS_KEY_ID")
	if s3AccessKeyID == "" {
		return nil, errors.New("no S3_ACCESS_KEY_ID found")
	}

	s3SecretAccessKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	if s3SecretAccessKey == "" {
		return nil, errors.New("no S3_SECRET_ACCESS_KEY found")
	}

	if os.Getenv("V2_SIGNING") == "true" {
		c, err = minio.NewV2(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	} else {
		c, err = minio.New(s3Endpoint, s3AccessKeyID, s3SecretAccessKey, true)
	}

	return c, err
}
