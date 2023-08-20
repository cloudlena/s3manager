package s3manager_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/cloudlena/s3manager/internal/s3manager"
	"github.com/cloudlena/s3manager/internal/s3manager/mocks"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleCreateObject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                 string
		putObjectErr       error
		bucketName         string
		path               string
		fileName           string
		expectedObjectName string
		expectedStatusCode int
		expectedHeaders    map[string]string
	}{
		{
			it:                 "creates a new object",
			bucketName:         "BUCKET-NAME",
			fileName:           "FILE-NAME.png",
			expectedObjectName: "FILE-NAME.png",
			expectedStatusCode: http.StatusCreated,
			expectedHeaders: map[string]string{
				"HX-Trigger": "objectListChanged",
			},
		},
		{
			it:                 "creates a new object with path",
			bucketName:         "BUCKET-NAME",
			path:               "TEST/PATH/",
			fileName:           "FILE-NAME.png",
			expectedObjectName: "TEST/PATH/FILE-NAME.png",
			expectedStatusCode: http.StatusCreated,
			expectedHeaders: map[string]string{
				"HX-Trigger": "objectListChanged",
			},
		},
		{
			it:                 "returns error for empty path",
			bucketName:         "",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			it:                 "returns error if there is an S3 error",
			putObjectErr:       errors.New("mocked S3 error"),
			bucketName:         "BUCKET-NAME",
			fileName:           "FILE-NAME.png",
			expectedObjectName: "FILE-NAME.png",
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				PutObjectFunc: func(ctx context.Context, bucketName, objectName string, file io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
					is.Equal(bucketName, tc.bucketName)         // bucket
					is.Equal(objectName, tc.expectedObjectName) // object
					return minio.UploadInfo{}, tc.putObjectErr
				},
			}
			server := s3manager.New(s3, true, "", "")

			engine := html.New("../../views", ".html.gotmpl")
			app := fiber.New(fiber.Config{
				Views: engine,
			})
			app.Post("/buckets/:bucket/objects", server.HandleCreateObject)

			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			pathField, err := writer.CreateFormField("path")
			is.NoErr(err)
			_, err = pathField.Write([]byte(tc.path))
			is.NoErr(err)
			part, err := writer.CreateFormFile("files", tc.fileName)
			is.NoErr(err)
			_, err = part.Write([]byte("FILE-CONTENT"))
			is.NoErr(err)
			writer.Close()
			req, err := http.NewRequest(http.MethodPost, "/buckets/BUCKET-NAME/objects", body)
			req.Header.Set(fiber.HeaderContentType, writer.FormDataContentType())
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
