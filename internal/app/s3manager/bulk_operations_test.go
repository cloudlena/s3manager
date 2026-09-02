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
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleBulkDeleteObjects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		removeObjectsFunc    func(context.Context, string, <-chan minio.ObjectInfo, minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "deletes multiple objects successfully",
			removeObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
				errCh := make(chan minio.RemoveObjectError)
				go func() {
					defer close(errCh)
					for range objectsCh {
					}
				}()
				return errCh
			},
			body:                 `{"keys":["file1.txt","file2.txt"]}`,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: `"success":true`,
		},
		{
			it: "returns error for invalid JSON body",
			removeObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
				errCh := make(chan minio.RemoveObjectError)
				close(errCh)
				return errCh
			},
			body:                 `not-json`,
			expectedStatusCode:   http.StatusUnprocessableEntity,
			expectedBodyContains: "error parsing request",
		},
		{
			it: "returns error when no keys provided",
			removeObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
				errCh := make(chan minio.RemoveObjectError)
				close(errCh)
				return errCh
			},
			body:                 `{"keys":[]}`,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "no keys provided",
		},
		{
			it: "returns error if S3 reports a remove error",
			removeObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
				errCh := make(chan minio.RemoveObjectError, 1)
				go func() {
					defer close(errCh)
					for range objectsCh {
					}
					errCh <- minio.RemoveObjectError{ObjectName: "file1.txt", Err: errS3}
				}()
				return errCh
			},
			body:                 `{"keys":["file1.txt"]}`,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				RemoveObjectsFunc: tc.removeObjectsFunc,
			}

			r := mux.NewRouter()
			r.Handle("/api/buckets/{bucketName}/objects/bulk-delete", s3manager.HandleBulkDeleteObjects(s3)).Methods(http.MethodPost)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/buckets/my-bucket/objects/bulk-delete", bytes.NewBufferString(tc.body))
			is.NoErr(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleBulkDownloadObjects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		keys                 string
		expectedStatusCode   int
		expectedBodyContains string
		expectedContentType  string
	}{
		{
			it:                   "returns bad request when no keys provided",
			keys:                 `[]`,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "no keys provided",
		},
		{
			it:                   "returns error for invalid keys JSON",
			keys:                 `not-json`,
			expectedStatusCode:   http.StatusUnprocessableEntity,
			expectedBodyContains: "error parsing keys",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{}

			r := mux.NewRouter()
			r.Handle("/api/buckets/{bucketName}/objects/bulk-download", s3manager.HandleBulkDownloadObjects(s3)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/api/buckets/my-bucket/objects/bulk-download?keys=" + tc.keys)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}
