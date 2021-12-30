package s3manager_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleBucketsView(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		listBucketsFunc      func(context.Context) ([]minio.BucketInfo, error)
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "renders a list of buckets",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{{Name: "testBucket"}}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "testBucket",
		},
		{
			it: "renders placeholder if no buckets",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		{
			it: "returns error if there is an S3 error",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{}, errS3
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListBucketsFunc: tc.listBucketsFunc,
			}

			templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))

			req, err := http.NewRequest(http.MethodGet, "/buckets", nil)
			is.NoErr(err)

			rr := httptest.NewRecorder()
			handler := s3manager.HandleBucketsView(s3, templates)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
