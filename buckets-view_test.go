package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestBucketsViewHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		s3                   S3Client
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"success": {
			s3: &S3ClientMock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "testBucket",
		},
		"success (bo buckets)": {
			s3:                   &S3ClientMock{},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("mocked S3 error"),
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range tests {
		req, err := http.NewRequest(http.MethodGet, "/buckets", nil)
		assert.NoError(err, tcID)

		rr := httptest.NewRecorder()
		handler := BucketsViewHandler(tc.s3)

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code, tcID)
		assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
	}
}
