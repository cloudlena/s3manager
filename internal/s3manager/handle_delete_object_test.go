package s3manager_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/cloudlena/s3manager/internal/s3manager"
	"github.com/cloudlena/s3manager/internal/s3manager/mocks"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleDeleteObject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                 string
		removeObjectErr    error
		bucketName         string
		objectName         string
		expectedStatusCode int
		expectedHeaders    map[string]string
	}{
		{
			it:                 "deletes an object",
			bucketName:         "BUCKET-NAME",
			objectName:         "OBJECT-NAME",
			expectedStatusCode: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"HX-Location": "/buckets/BUCKET-NAME",
			},
		},
		{
			it:                 "deletes a nested object",
			bucketName:         "BUCKET-NAME",
			objectName:         "OBJECT-PATH/OBJECT-NAME",
			expectedStatusCode: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"HX-Location": "/buckets/BUCKET-NAME",
			},
		},
		{
			it:                 "returns error if there is an S3 error",
			removeObjectErr:    errors.New("mocked S3 error"),
			bucketName:         "BUCKET-NAME",
			objectName:         "OBJECT-NAME",
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				RemoveObjectFunc: func(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error {
					is.Equal(bucketName, tc.bucketName) // bucket
					is.Equal(objectName, tc.objectName) // object
					return tc.removeObjectErr
				},
			}
			server := s3manager.New(s3, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Delete("/buckets/:bucket/objects/+", server.HandleDeleteObject)

			req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/buckets/%s/objects/%s", tc.bucketName, tc.objectName), nil)
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
