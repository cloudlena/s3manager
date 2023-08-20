package s3manager_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudlena/s3manager/internal/s3manager"
	"github.com/cloudlena/s3manager/internal/s3manager/mocks"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/matryer/is"
)

func TestHandleDeleteBucket(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                 string
		removeBucketErr    error
		bucketName         string
		expectedStatusCode int
		expectedHeaders    map[string]string
	}{
		{
			it:                 "deletes a bucket",
			bucketName:         "BUCKET-NAME",
			expectedStatusCode: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"HX-Location": "/buckets",
			},
		},
		{
			it:                 "returns error if there is an S3 error",
			removeBucketErr:    errors.New("mocked S3 error"),
			bucketName:         "BUCKET-NAME",
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				RemoveBucketFunc: func(ctx context.Context, bucketName string) error {
					is.Equal(bucketName, tc.bucketName) // bucket
					return tc.removeBucketErr
				},
			}
			server := s3manager.New(s3, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Delete("/buckets/:bucket", server.HandleDeleteBucket)

			req, err := http.NewRequest(http.MethodDelete, "/buckets/"+tc.bucketName, nil)
			is.NoErr(err)

			resp, err := app.Test(req)
			is.NoErr(err)

			is.Equal(resp.StatusCode, tc.expectedStatusCode) // status code
			for k, v := range tc.expectedHeaders {
				is.Equal(resp.Header.Get(k), v) // header
			}
		})
	}
}
