package s3manager

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

//go:generate moq -out mocks/s3.go -pkg mocks . S3

// S3 is a client to interact with S3 storage.
type S3 interface {
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	RemoveBucket(ctx context.Context, bucketName string) error
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	GetBucketPolicy(ctx context.Context, bucketName string) (string, error)
	SetBucketPolicy(ctx context.Context, bucketName string, policy string) error
	RemoveObjects(ctx context.Context, bucketName string, objectsCh <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
	EndpointURL() *url.URL
}

// SSEType describes a type of server side encryption.
type SSEType struct {
	Type string
	Key  string
}

// Options holds the features and presentation settings the handlers share.
type Options struct {
	// RootURL is the path prefix the app is served under, for reverse proxying.
	RootURL string
	// BucketName restricts the bucket list to a single bucket if set.
	BucketName string
	// AllowDelete enables the delete actions.
	AllowDelete bool
	// ForceDownload serves objects as attachments instead of letting the
	// browser decide.
	ForceDownload bool
	// ListRecursive lists a bucket's objects across all its prefixes.
	ListRecursive bool
	// ShowVersions lists and serves all versions of an object.
	ShowVersions bool
	// ShowMetadata enables the object metadata endpoint and action.
	ShowMetadata bool
	// SSE is the server side encryption applied to uploads.
	SSE SSEType
}
