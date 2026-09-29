package s3manager_test

import (
	"errors"
	"net/url"

	"github.com/minio/minio-go/v7"
)

var (
	errS3                 = errors.New("mocked s3 error")
	errBucketDoesNotExist = minio.ErrorResponse{Code: minio.NoSuchBucket, Message: "The specified bucket does not exist"}
	// errV2Unpageable is what minio-go reports for a truncated ListObjects V2
	// page without a continuation token, as older Ceph RGW releases send.
	errV2Unpageable = minio.ErrorResponse{Code: minio.NotImplemented, Message: "Truncated response should have continuation token set"}
)

// mustParseURLFunc returns a mock's EndpointURL implementation for a fixed URL.
func mustParseURLFunc(rawURL string) func() *url.URL {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return func() *url.URL { return parsed }
}
