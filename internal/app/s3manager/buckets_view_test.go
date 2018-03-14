package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mastertinner/s3manager/internal/app/s3manager"
	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestBucketsViewHandler(t *testing.T) {
	cases := map[string]struct {
		s3                   s3manager.S3
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"renders a list of buckets": {
			s3: &s3Mock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "testBucket",
		},
		"renders placeholder if no buckets": {
			s3:                   &s3Mock{},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		"returns error if there is an S3 error": {
			s3: &s3Mock{
				Err: errors.New("mocked S3 error"),
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			assert := assert.New(t)

			tmplDir := filepath.Join("..", "..", "..", "web", "template")
			r := mux.NewRouter()
			r.
				Methods(http.MethodGet).
				Path("/buckets/{bucketName}").
				Handler(s3manager.BucketViewHandler(tc.s3, tmplDir))

			req, err := http.NewRequest(http.MethodGet, "/buckets", nil)
			assert.NoError(err, tcID)

			rr := httptest.NewRecorder()
			handler := s3manager.BucketsViewHandler(tc.s3, tmplDir)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()

			assert.Equal(tc.expectedStatusCode, resp.StatusCode, tcID)
			assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
		})
	}
}
