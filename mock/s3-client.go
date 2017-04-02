package mock

import (
	"io"

	minio "github.com/minio/minio-go"
)

type S3Client struct {
	Buckets     []minio.BucketInfo
	ObjectInfos []minio.ObjectInfo
	Objects     []minio.Object
	Err         error
}

func (s S3Client) CopyObject(string, string, string, minio.CopyConditions) error {
	return s.Err
}

func (s S3Client) GetObject(string, string) (*minio.Object, error) {
	return &s.Objects[0], s.Err
}

func (s S3Client) ListBuckets() ([]minio.BucketInfo, error) {
	return s.Buckets, s.Err
}

func (s S3Client) ListObjectsV2(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
	return make(<-chan minio.ObjectInfo)
}

func (s S3Client) MakeBucket(string, string) error {
	return s.Err
}

func (s S3Client) PutObject(string, string, io.Reader, string) (int64, error) {
	return 0, s.Err
}

func (s S3Client) RemoveBucket(string) error {
	return s.Err
}

func (s S3Client) RemoveObject(string, string) error {
	return s.Err
}
