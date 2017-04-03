package main

import (
	"io"

	minio "github.com/minio/minio-go"
)

// S3Client is a client to interact with S3 storage
type S3Client interface {
	CopyObject(string, string, string, minio.CopyConditions) error
	GetObject(string, string) (*minio.Object, error)
	ListBuckets() ([]minio.BucketInfo, error)
	ListObjectsV2(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo
	MakeBucket(string, string) error
	PutObject(string, string, io.Reader, string) (int64, error)
	RemoveBucket(string) error
	RemoveObject(string, string) error
}
