package s3manager_test

import (
	"io"

	. "github.com/mastertinner/s3manager"
	minio "github.com/minio/minio-go"
)

// s3Mock is a mocked S3 client.
type s3Mock struct {
	Buckets []minio.BucketInfo
	Objects []minio.ObjectInfo
	Err     error
}

// CopyObject mocks minio.Client.CopyObject.
func (s *s3Mock) CopyObject(string, string, string, minio.CopyConditions) error {
	return s.Err
}

// GetObject mocks minio.Client.GetObject.
func (s *s3Mock) GetObject(bucketName string, objectName string) (*minio.Object, error) {
	if s.Err != nil {
		return nil, s.Err
	}

	return &minio.Object{}, nil
}

// ListBuckets mocks minio.Client.ListBuckets.
func (s *s3Mock) ListBuckets() ([]minio.BucketInfo, error) {
	return s.Buckets, s.Err
}

// ListObjectsV2 mocks minio.Client.ListObjectsV2.
func (s *s3Mock) ListObjectsV2(name string, p string, r bool, d <-chan struct{}) <-chan minio.ObjectInfo {
	// Add error if exists
	if s.Err != nil {
		s.Objects = append(s.Objects, minio.ObjectInfo{
			Err: s.Err,
		})
	}

	// Check if bucket exists
	found := false
	for _, b := range s.Buckets {
		if b.Name == name {
			found = true
			break
		}
	}
	if !found {
		s.Objects = append(s.Objects, minio.ObjectInfo{
			Err: ErrBucketDoesNotExist,
		})

	}

	objCh := make(chan minio.ObjectInfo, len(s.Objects))
	defer close(objCh)

	for _, obj := range s.Objects {
		objCh <- obj
	}

	return objCh
}

// MakeBucket mocks minio.Client.MakeBucket.
func (s *s3Mock) MakeBucket(string, string) error {
	return s.Err
}

// PutObject mocks minio.Client.PutObject.
func (s *s3Mock) PutObject(string, string, io.Reader, string) (int64, error) {
	return 0, s.Err
}

// RemoveBucket mocks minio.Client.RemoveBucket.
func (s *s3Mock) RemoveBucket(string) error {
	return s.Err
}

// RemoveObject mocks minio.Client.RemoveObject.
func (s *s3Mock) RemoveObject(string, string) error {
	return s.Err
}
