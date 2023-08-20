package s3manager_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/s3manager"
	"github.com/cloudlena/s3manager/internal/s3manager/mocks"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleCreateBucket(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                 string
		makeBucketErr      error
		bucketName         string
		expectedStatusCode int
		expectedHeaders    map[string]string
	}{
		{
			it:                 "creates a new bucket",
			bucketName:         "BUCKET-NAME",
			expectedStatusCode: http.StatusCreated,
			expectedHeaders: map[string]string{
				"HX-Trigger": "bucketListChanged",
			},
		},
		{
			it:                 "returns error for empty request",
			bucketName:         "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			it:                 "returns error if there is an S3 error",
			makeBucketErr:      errors.New("mocked S3 error"),
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
				MakeBucketFunc: func(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
					is.Equal(bucketName, tc.bucketName) // bucket
					return tc.makeBucketErr
				},
			}
			server := s3manager.New(s3, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Post("/buckets", server.HandleCreateBucket)

			form := url.Values{}
			form.Add("name", tc.bucketName)
			req, err := http.NewRequest(http.MethodPost, "/buckets", strings.NewReader(form.Encode()))
			req.Header.Add(fiber.HeaderContentType, fiber.MIMEApplicationForm)
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
