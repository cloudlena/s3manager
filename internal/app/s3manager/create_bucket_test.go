package s3manager_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleCreateBucket(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		makeBucketFunc       func(context.Context, string, minio.MakeBucketOptions) error
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "creates a new bucket",
			makeBucketFunc: func(context.Context, string, minio.MakeBucketOptions) error {
				return nil
			},
			body:                 `{"name":"myBucket"}`,
			expectedStatusCode:   http.StatusCreated,
			expectedBodyContains: `{"name":"myBucket","creationDate":"0001-01-01T00:00:00Z"}`,
		},
		{
			it: "returns error for empty request",
			makeBucketFunc: func(context.Context, string, minio.MakeBucketOptions) error {
				return nil
			},
			body:                 "",
			expectedStatusCode:   http.StatusUnprocessableEntity,
			expectedBodyContains: http.StatusText(http.StatusUnprocessableEntity),
		},
		{
			it: "returns error for malformed request",
			makeBucketFunc: func(context.Context, string, minio.MakeBucketOptions) error {
				return nil
			},
			body:                 "}",
			expectedStatusCode:   http.StatusUnprocessableEntity,
			expectedBodyContains: http.StatusText(http.StatusUnprocessableEntity),
		},
		{
			it: "returns error if there is an S3 error",
			makeBucketFunc: func(context.Context, string, minio.MakeBucketOptions) error {
				return errS3
			},
			body:                 `{"name":"myBucket"}`,
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
				MakeBucketFunc: tc.makeBucketFunc,
			}

			req, err := http.NewRequest(http.MethodPost, "/api/buckets", bytes.NewBufferString(tc.body))
			is.NoErr(err)

			rr := httptest.NewRecorder()
			handler := s3manager.HandleCreateBucket(s3)

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
