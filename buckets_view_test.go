package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/mastertinner/s3manager"
	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestBucketsViewHandler(t *testing.T) {
	assert := assert.New(t)

	cases := map[string]struct {
		s3                   S3
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"success": {
			s3: &s3Mock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "testBucket",
		},
		"success (bo buckets)": {
			s3:                   &s3Mock{},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		"s3 error": {
			s3: &s3Mock{
				Err: errors.New("mocked S3 error"),
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/buckets", nil)
			assert.NoError(err, tcID)

			rr := httptest.NewRecorder()
			handler := BucketsViewHandler(tc.s3)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()

			assert.Equal(tc.expectedStatusCode, resp.StatusCode, tcID)
			assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
		})
	}
}
