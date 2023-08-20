package s3manager_test

import (
	"context"
	"errors"
	"fmt"
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

func TestHandleObjectList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		listObjectsVal       []minio.ObjectInfo
		bucketName           string
		path                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:                   "renders a bucket containing a file",
			listObjectsVal:       []minio.ObjectInfo{{Key: "FILE-NAME"}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "FILE-NAME",
		},
		{
			it:                   "renders placeholder for an empty bucket",
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No objects in",
		},
		{
			it:                   "renders a bucket containing an archive",
			listObjectsVal:       []minio.ObjectInfo{{Key: "archive.tar.gz"}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "archive",
		},
		{
			it:                   "renders a bucket containing an image",
			listObjectsVal:       []minio.ObjectInfo{{Key: "FILE-NAME.png"}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "photo",
		},
		{
			it:                   "renders a bucket containing a sound file",
			listObjectsVal:       []minio.ObjectInfo{{Key: "FILE-NAME.mp3"}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "music_note",
		},
		{
			it:                   "returns error if the bucket doesn't exist",
			listObjectsVal:       []minio.ObjectInfo{{Err: errors.New("error: The specified bucket does not exist")}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "error: The specified bucket does not exist",
		},
		{
			it:                   "returns error if there is an S3 error",
			listObjectsVal:       []minio.ObjectInfo{{Err: errors.New("mocked s3 error")}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
		{
			it:                   "renders a bucket with folder",
			listObjectsVal:       []minio.ObjectInfo{{Key: "TEST-FOLDER/"}},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "folder",
		},
		{
			it:                   "renders a bucket with path",
			bucketName:           "BUCKET-NAME",
			path:                 "abc/def",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "def",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListObjectsFunc: func(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
					is.Equal(bucketName, tc.bucketName) // bucket
					objCh := make(chan minio.ObjectInfo)
					go func() {
						for _, object := range tc.listObjectsVal {
							objCh <- object
						}
						close(objCh)
					}()
					return objCh
				},
			}
			server := s3manager.New(s3, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Get("/buckets/:bucket/object-list/*", server.HandleObjectList)

			req, err := http.NewRequest(fiber.MethodGet, fmt.Sprintf("/buckets/%s/object-list/%s", tc.bucketName, tc.path), nil)
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
