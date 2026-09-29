package s3manager_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
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
			body:               `{"keys":["file1.txt","file2.txt"]}`,
			expectedStatusCode: http.StatusNoContent,
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

func TestHandleBulkDeleteObjectsWithFolders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                 string
		body               string
		folders            map[string][]string
		v1Folders          map[string][]string
		listErr            error
		v2ListErr          error
		v1ListErr          error
		expectedStatusCode int
		expectedRemoved    []string
	}{
		{
			it:                 "deletes everything inside a folder",
			body:               `{"keys":["photos/"]}`,
			folders:            map[string][]string{"photos/": {"photos/", "photos/a.jpg", "photos/2024/b.jpg"}},
			expectedStatusCode: http.StatusNoContent,
			expectedRemoved:    []string{"photos/", "photos/a.jpg", "photos/2024/b.jpg"},
		},
		{
			it:                 "deletes a mix of objects and folders",
			body:               `{"keys":["readme.txt","photos/","docs/"]}`,
			folders:            map[string][]string{"photos/": {"photos/a.jpg"}, "docs/": {"docs/b.pdf"}},
			expectedStatusCode: http.StatusNoContent,
			expectedRemoved:    []string{"readme.txt", "photos/a.jpg", "docs/b.pdf"},
		},
		{
			it:                 "falls back to a V1 listing for a folder that lists empty",
			body:               `{"keys":["photos/"]}`,
			v1Folders:          map[string][]string{"photos/": {"photos/a.jpg"}},
			expectedStatusCode: http.StatusNoContent,
			expectedRemoved:    []string{"photos/a.jpg"},
		},
		{
			it:                 "falls back to a V1 listing for a folder whose V2 listing can't be paged",
			body:               `{"keys":["photos/"]}`,
			v1Folders:          map[string][]string{"photos/": {"photos/a.jpg"}},
			v2ListErr:          errV2Unpageable,
			expectedStatusCode: http.StatusNoContent,
			expectedRemoved:    []string{"photos/a.jpg"},
		},
		{
			it:                 "resumes the V1 listing after the objects V2 listed before failing",
			body:               `{"keys":["photos/"]}`,
			folders:            map[string][]string{"photos/": {"photos/a.jpg"}},
			v1Folders:          map[string][]string{"photos/": {"photos/a.jpg", "photos/b.jpg"}},
			v2ListErr:          errV2Unpageable,
			expectedStatusCode: http.StatusNoContent,
			expectedRemoved:    []string{"photos/a.jpg", "photos/b.jpg"},
		},
		{
			it:                 "returns error if the V1 fallback for an unpageable folder fails",
			body:               `{"keys":["photos/"]}`,
			v2ListErr:          errV2Unpageable,
			v1ListErr:          errS3,
			expectedStatusCode: http.StatusInternalServerError,
			expectedRemoved:    nil,
		},
		{
			it:                 "deletes nothing for an empty folder",
			body:               `{"keys":["photos/"]}`,
			expectedStatusCode: http.StatusNoContent,
			expectedRemoved:    nil,
		},
		{
			it:                 "returns error if listing a folder fails",
			body:               `{"keys":["readme.txt","photos/"]}`,
			listErr:            errS3,
			expectedStatusCode: http.StatusInternalServerError,
			expectedRemoved:    []string{"readme.txt"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			var removed []string
			s3 := &mocks.S3Mock{
				ListObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
					is.True(opts.Recursive)
					folders, trailingErr := tc.folders, tc.v2ListErr
					if opts.UseV1 {
						folders, trailingErr = tc.v1Folders, tc.v1ListErr
					}
					objCh := make(chan minio.ObjectInfo, len(folders[opts.Prefix])+2)
					if tc.listErr != nil {
						objCh <- minio.ObjectInfo{Err: tc.listErr}
					}
					for _, key := range folders[opts.Prefix] {
						if key > opts.StartAfter {
							objCh <- minio.ObjectInfo{Key: key}
						}
					}
					if trailingErr != nil {
						objCh <- minio.ObjectInfo{Err: trailingErr}
					}
					close(objCh)
					return objCh
				},
				RemoveObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
					errCh := make(chan minio.RemoveObjectError)
					go func() {
						defer close(errCh)
						for object := range objectsCh {
							removed = append(removed, object.Key)
						}
					}()
					return errCh
				},
			}

			r := mux.NewRouter()
			r.Handle("/api/buckets/{bucketName}/objects/bulk-delete", s3manager.HandleBulkDeleteObjects(s3)).Methods(http.MethodPost)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Post(ts.URL+"/api/buckets/my-bucket/objects/bulk-delete", "application/json", bytes.NewBufferString(tc.body))
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.Equal(tc.expectedRemoved, removed)
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
			r.Handle("/api/buckets/{bucketName}/objects/bulk-download", s3manager.HandleBulkDownloadObjects(s3, nil)).Methods(http.MethodGet)

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

func TestHandleBulkDownloadObjectsWithFolders(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s3 := &mocks.S3Mock{
		ListObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
			objCh := make(chan minio.ObjectInfo, 2)
			if opts.Prefix == "photos/" && opts.Recursive {
				objCh <- minio.ObjectInfo{Key: "photos/a.jpg"}
				objCh <- minio.ObjectInfo{Key: "photos/2024/b.jpg"}
			}
			close(objCh)
			return objCh
		},
		GetObjectFunc: func(context.Context, string, string, minio.GetObjectOptions) (*minio.Object, error) {
			return nil, errS3
		},
	}

	// SSE-C objects can only be read with the key they were stored with.
	sse, err := encrypt.NewSSEC([]byte("0123456789abcdef0123456789abcdef"))
	is.NoErr(err)

	r := mux.NewRouter()
	r.Handle("/api/buckets/{bucketName}/objects/bulk-download", s3manager.HandleBulkDownloadObjects(s3, sse)).Methods(http.MethodPost)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.PostForm(ts.URL+"/api/buckets/my-bucket/objects/bulk-download", url.Values{"keys": {`["readme.txt","photos/"]`}})
	is.NoErr(err)
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()

	is.Equal(http.StatusOK, resp.StatusCode)

	var fetched []string
	for _, call := range s3.GetObjectCalls() {
		fetched = append(fetched, call.ObjectName)
		is.Equal(sse, call.Opts.ServerSideEncryption) // encryption key sent along
	}
	is.Equal([]string{"readme.txt", "photos/a.jpg", "photos/2024/b.jpg"}, fetched)
}
