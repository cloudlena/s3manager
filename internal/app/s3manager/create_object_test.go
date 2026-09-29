package s3manager_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleCreateObject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		putObjectFunc        func(context.Context, string, string, io.Reader, int64, minio.PutObjectOptions) (minio.UploadInfo, error)
		fileName             string
		filePath             string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "uploads an object successfully",
			putObjectFunc: func(context.Context, string, string, io.Reader, int64, minio.PutObjectOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, nil
			},
			fileName:           "test.txt",
			filePath:           "test.txt",
			expectedStatusCode: http.StatusCreated,
		},
		{
			it: "uploads with explicit path",
			putObjectFunc: func(context.Context, string, string, io.Reader, int64, minio.PutObjectOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, nil
			},
			fileName:           "test.txt",
			filePath:           "folder/test.txt",
			expectedStatusCode: http.StatusCreated,
		},
		{
			it: "returns error if there is an S3 error",
			putObjectFunc: func(context.Context, string, string, io.Reader, int64, minio.PutObjectOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, errS3
			},
			fileName:             "test.txt",
			filePath:             "test.txt",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				PutObjectFunc: tc.putObjectFunc,
			}

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", tc.fileName)
			is.NoErr(err)
			_, err = io.Copy(part, strings.NewReader("file content"))
			is.NoErr(err)
			err = writer.WriteField("path", tc.filePath)
			is.NoErr(err)
			err = writer.Close()
			is.NoErr(err)

			r := mux.NewRouter()
			r.Handle("/api/buckets/{bucketName}/objects", s3manager.HandleCreateObject(s3, nil)).Methods(http.MethodPost)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/buckets/my-bucket/objects", &body)
			is.NoErr(err)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			respBody, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(respBody), tc.expectedBodyContains))
		})
	}
}

func TestHandleCreateObjectInvalidRequest(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s3 := &mocks.S3Mock{}

	r := mux.NewRouter()
	r.Handle("/api/buckets/{bucketName}/objects", s3manager.HandleCreateObject(s3, nil)).Methods(http.MethodPost)

	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/buckets/my-bucket/objects", strings.NewReader("not-a-multipart-form"))
	is.NoErr(err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	is.NoErr(err)
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()

	is.Equal(http.StatusInternalServerError, resp.StatusCode)
}
