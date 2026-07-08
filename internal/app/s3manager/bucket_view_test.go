package s3manager_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleBucketView(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		listObjectsFunc      func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo
		bucketName           string
		rootUrl              string
		path                 string
		showVersions         bool
		expectedStatusCode   int
		expectedBodyContains string
		unexpectedInBody     []string
	}{
		{
			it: "renders a bucket containing a file",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "FILE-NAME"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "FILE-NAME",
		},
		{
			it: "renders placeholder for an empty bucket",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				close(objCh)
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No objects in",
		},
		{
			it: "renders a bucket containing an archive",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "archive.tar.gz"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "archive",
		},
		{
			it: "renders a bucket containing an image",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "FILE-NAME.png"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "photo",
		},
		{
			it: "renders a bucket containing a sound file",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "FILE-NAME.mp3"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "music_note",
		},
		{
			it: "returns error if the bucket doesn't exist",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Err: errBucketDoesNotExist}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "bucket does not exist",
		},
		{
			it: "returns error if there is an S3 error",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Err: errS3}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
		{
			it: "renders a bucket with folder",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "AFolder/"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "folder",
		},
		{
			it: "renders a bucket with path",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				close(objCh)
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			path:                 "abc/def",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "def",
		},
		{
			it: "renders a bucket with path",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				close(objCh)
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			path:                 "abc/def",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "def",
		},
		{
			it: "setting rootUrl works",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				close(objCh)
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			path:                 "abc/def",
			rootUrl:              "rootTest",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "def",
		},
		{
			it: "does not show version columns when ShowVersions is disabled",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk", IsLatest: true}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			showVersions:         false,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "FILE-NAME",
			unexpectedInBody:     []string{"Version ID", "v1-abcdef"},
		},
		{
			it: "renders multiple versions when ShowVersions is enabled",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					if opts.WithVersions {
						objCh <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v2-abcdefghijk", IsLatest: true}
						objCh <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk", IsLatest: false}
					}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "Latest",
		},
		{
			it: "falls back to a normal listing when the versioned listing fails",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					defer close(objCh)
					if opts.WithVersions {
						objCh <- minio.ObjectInfo{Err: errS3}
						return
					}
					objCh <- minio.ObjectInfo{Key: "FILE-NAME"}
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "FILE-NAME",
			unexpectedInBody:     []string{"Version ID"},
		},
		{
			it: "falls back to a normal listing when the versioned listing succeeds but returns nothing",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					defer close(objCh)
					if opts.WithVersions {
						return
					}
					objCh <- minio.ObjectInfo{Key: "FILE-NAME"}
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "FILE-NAME",
			unexpectedInBody:     []string{"Version ID"},
		},
		{
			it: "collapses older versions by default with a toggle to expand them",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					if opts.WithVersions {
						objCh <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v2-abcdefghijk", IsLatest: true}
						objCh <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk", IsLatest: false}
					}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: `class="version-row" style="display: none;`,
		},
		{
			it: "does not hide folders or objects when the provider never sets IsLatest on versioned entries",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					defer close(objCh)
					if !opts.WithVersions {
						return
					}
					// Folders synthesized from CommonPrefixes never carry
					// version metadata, and some providers don't reliably
					// set IsLatest on real objects either.
					objCh <- minio.ObjectInfo{Key: "AFolder/"}
					objCh <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk"}
				}()
				return objCh
			},
			bucketName:           "BUCKET-NAME",
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "AFolder",
			unexpectedInBody:     []string{`class="version-row" style="display: none;`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListObjectsFunc: tc.listObjectsFunc,
				EndpointURLFunc: func() *url.URL {
					u, _ := url.Parse("http://localhost:9000")
					return u
				},
			}

			templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))
			r := mux.NewRouter()
			r.PathPrefix("/buckets/").Handler(s3manager.HandleBucketView(s3, templates, true, true, tc.rootUrl, tc.showVersions)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(fmt.Sprintf("%s/buckets/%s/%s", ts.URL, tc.bucketName, tc.path))
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
			for _, unexpected := range tc.unexpectedInBody {
				is.True(!strings.Contains(string(body), unexpected))
			}

			// fmt.Println(string(body))
			if tc.expectedStatusCode == http.StatusOK {
				hyperlink := fmt.Sprintf("<a href=\"%s/buckets\" class=\"breadcrumb\"><i class=\"material-icons\">arrow_back</i> buckets </a>", tc.rootUrl)
				is.True(strings.Contains(string(body), hyperlink))
			}
		})
	}
}
