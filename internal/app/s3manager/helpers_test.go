package s3manager_test

import (
	"errors"
	"net/url"
)

var (
	errS3                 = errors.New("mocked s3 error")
	errBucketDoesNotExist = errors.New("error: The specified bucket does not exist")
)

// mustParseURLFunc returns a mock's EndpointURL implementation for a fixed URL.
func mustParseURLFunc(rawURL string) func() *url.URL {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return func() *url.URL { return parsed }
}
