package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestHandleBucketsView(t *testing.T) {
	cases := map[string]struct {
		listBucketsFunc      func() ([]minio.BucketInfo, error)
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"renders a list of buckets": {
			listBucketsFunc: func() ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{{Name: "testBucket"}}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "testBucket",
		},
		"renders placeholder if no buckets": {
			listBucketsFunc: func() ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		"returns error if there is an S3 error": {
			listBucketsFunc: func() ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{}, errors.New("mocked S3 error")
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			assert := assert.New(t)

			s3 := &S3Mock{
				ListBucketsFunc: tc.listBucketsFunc,
			}

			tmplDir := filepath.Join("..", "..", "..", "web", "template")

			req, err := http.NewRequest(http.MethodGet, "/buckets", nil)
			assert.NoError(err)

			rr := httptest.NewRecorder()
			handler := s3manager.HandleBucketsView(s3, tmplDir)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()

			assert.Equal(tc.expectedStatusCode, resp.StatusCode)
			assert.Contains(rr.Body.String(), tc.expectedBodyContains)
		})
	}
}
