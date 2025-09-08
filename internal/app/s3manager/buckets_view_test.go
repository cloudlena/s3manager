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
				return []minio.BucketInfo{{Name: "BUCKET-NAME"}}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "BUCKET-NAME",
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
			handler := s3manager.HandleBucketsView(s3, templates, true, "#ee6e73", "/static/img/logo.png", "#f44336")

			handler.ServeHTTP(rr, req)
			resp := rr.Result()
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
