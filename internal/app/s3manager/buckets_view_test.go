package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/mastertinner/s3manager/internal/app/s3manager/mocks"
	"github.com/matryer/is"
	minio "github.com/minio/minio-go"
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
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListBucketsFunc: tc.listBucketsFunc,
			}

			tmplDir := filepath.Join("..", "..", "..", "web", "template")

			req, err := http.NewRequest(http.MethodGet, "/buckets", nil)
			is.NoErr(err)

			rr := httptest.NewRecorder()
			handler := s3manager.HandleBucketsView(s3, tmplDir)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                     // status code
			is.True(strings.Contains(rr.Body.String(), tc.expectedBodyContains)) // body
		})
	}
}
