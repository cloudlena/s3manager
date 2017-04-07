package main

import (
	"errors"
	"io"

	minio "github.com/minio/minio-go"
)

// S3ClientMock is a mocked S3 client
type S3ClientMock struct {
	Buckets []minio.BucketInfo
	Objects []minio.ObjectInfo
	Err     error
}

// CopyObject mocks minio.Client.CopyObject
func (s S3ClientMock) CopyObject(string, string, string, minio.CopyConditions) error {
	return s.Err
}

// GetObject mocks minio.Client.GetObject
func (s S3ClientMock) GetObject(bucketName string, objectName string) (*minio.Object, error) {
	if s.Err != nil {
		return nil, s.Err
	}

	return &minio.Object{}, nil
}

// ListBuckets mocks minio.Client.ListBuckets
func (s S3ClientMock) ListBuckets() ([]minio.BucketInfo, error) {
	return s.Buckets, s.Err
}

// ListObjectsV2 mocks minio.Client.ListObjectsV2
func (s S3ClientMock) ListObjectsV2(bucketName string, p string, r bool, d <-chan struct{}) <-chan minio.ObjectInfo {
	// Add error if exists
	if s.Err != nil {
		s.Objects = append(s.Objects, minio.ObjectInfo{
			Err: s.Err,
		})
	}

	// Check if bucket exists
	found := false
	for _, b := range s.Buckets {
		if b.Name == bucketName {
			found = true
			break
		}
	}
	if !found {
		s.Objects = append(s.Objects, minio.ObjectInfo{
			Err: errors.New("The specified bucket does not exist."),
		})

	}

	objCh := make(chan minio.ObjectInfo, len(s.Objects))
	defer close(objCh)

	for _, obj := range s.Objects {
		objCh <- obj
	}

	return objCh
}

// MakeBucket mocks minio.Client.MakeBucket
func (s S3ClientMock) MakeBucket(string, string) error {
	return s.Err
}

// PutObject mocks minio.Client.PutObject
func (s S3ClientMock) PutObject(string, string, io.Reader, string) (int64, error) {
	return 0, s.Err
}

// RemoveBucket mocks minio.Client.RemoveBucket
func (s S3ClientMock) RemoveBucket(string) error {
	return s.Err
}

// RemoveObject mocks minio.Client.RemoveObject
func (s S3ClientMock) RemoveObject(string, string) error {
	return s.Err
}
