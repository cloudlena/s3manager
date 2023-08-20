package s3manager_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/s3manager"
	"github.com/cloudlena/s3manager/internal/s3manager/mocks"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleBucketList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		listBucketsVal       []minio.BucketInfo
		listBucketsErr       error
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:                   "renders a list of buckets",
			listBucketsVal:       []minio.BucketInfo{{Name: "BUCKET-NAME"}},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "BUCKET-NAME",
		},
		{
			it:                   "renders placeholder if no buckets",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No buckets yet",
		},
		{
			it:                   "returns error if there is an S3 error",
			listBucketsErr:       errors.New("mocked s3 error"),
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListBucketsFunc: func(ctx context.Context) ([]minio.BucketInfo, error) {
					return tc.listBucketsVal, tc.listBucketsErr
				},
			}
			server := s3manager.New(s3, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Get("/bucket-list", server.HandleBucketList)

			req, err := http.NewRequest(fiber.MethodGet, "/bucket-list", nil)
			is.NoErr(err)

			resp, err := app.Test(req)
			is.NoErr(err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(resp.StatusCode, tc.expectedStatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
