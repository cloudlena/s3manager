package main

import (
	"io"

	minio "github.com/minio/minio-go"
)

// S3ClientMock is a mocked S3 client
type S3ClientMock struct {
	Buckets     []minio.BucketInfo
	ObjectInfos []minio.ObjectInfo
	Objects     []minio.Object
	Err         error
}

func (s S3ClientMock) CopyObject(string, string, string, minio.CopyConditions) error {
	return s.Err
}

func (s S3ClientMock) GetObject(string, string) (*minio.Object, error) {
	return &s.Objects[0], s.Err
}

func (s S3ClientMock) ListBuckets() ([]minio.BucketInfo, error) {
	return s.Buckets, s.Err
}

func (s S3ClientMock) ListObjectsV2(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
	return make(<-chan minio.ObjectInfo)
}

func (s S3ClientMock) MakeBucket(string, string) error {
	return s.Err
}

func (s S3ClientMock) PutObject(string, string, io.Reader, string) (int64, error) {
	return 0, s.Err
}

func (s S3ClientMock) RemoveBucket(string) error {
	return s.Err
}

func (s S3ClientMock) RemoveObject(string, string) error {
	return s.Err
}
