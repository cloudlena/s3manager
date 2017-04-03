package main

import (
	"errors"
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

func (s S3ClientMock) ListObjectsV2(bucketName string, p string, r bool, d <-chan struct{}) <-chan minio.ObjectInfo {
	// Add error if exists
	if s.Err != nil {
		s.ObjectInfos = append(s.ObjectInfos, minio.ObjectInfo{
			Err: s.Err,
		})
	}

	// Check if bucket exists
	found := false
	for _, b := range s.Buckets {
		if b.Name == bucketName {
			found = true
		}
	}
	if !found {
		s.ObjectInfos = append(s.ObjectInfos, minio.ObjectInfo{
			Err: errors.New("The specified bucket does not exist."),
		})

	}

	objCh := make(chan minio.ObjectInfo, len(s.ObjectInfos))
	defer close(objCh)

	for _, obj := range s.ObjectInfos {
		objCh <- obj
	}

	return objCh
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
