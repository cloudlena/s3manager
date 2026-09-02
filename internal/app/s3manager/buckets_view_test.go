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
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleBucketsView(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		bucketName           string
		listBucketsFunc      func(context.Context) ([]minio.BucketInfo, error)
		expectedStatusCode   int
		expectedBodyContains string
		unexpectedInBody     []string
	}{
		{
			it:           "renders a list of buckets",
			instanceName: "primary",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{{Name: "BUCKET-NAME"}}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "BUCKET-NAME",
		},
		{
			it:           "renders placeholder if no buckets",
			instanceName: "primary",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		{
			it:           "only renders the configured bucket",
			instanceName: "primary",
			bucketName:   "BUCKET-NAME",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{{Name: "BUCKET-NAME"}, {Name: "OTHER-BUCKET"}}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "BUCKET-NAME",
			unexpectedInBody:     []string{"OTHER-BUCKET"},
		},
		{
			it:           "shows an error message if the instance is unreachable",
			instanceName: "primary",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return nil, errS3
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "Unable to connect to S3 instance",
		},
		{
			it:           "returns not found for an unknown instance",
			instanceName: "unknown",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return nil, nil
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{ListBucketsFunc: tc.listBucketsFunc}
			instances := s3manager.S3Instances{{ID: "1", Name: "primary", Client: s3}}
			templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))

			r := mux.NewRouter()
			r.Handle("/{instance}/buckets", s3manager.HandleBucketsView(instances, templates, s3manager.Options{
				AllowDelete: true,
				BucketName:  tc.bucketName,
			})).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/buckets")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
			for _, unexpected := range tc.unexpectedInBody {
				is.True(!strings.Contains(string(body), unexpected)) // unexpected body content
			}
		})
	}
}
