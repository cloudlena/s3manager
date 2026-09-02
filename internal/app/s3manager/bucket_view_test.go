package s3manager_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

// objectChan serves the given objects on a ListObjects channel.
func objectChan(objects ...minio.ObjectInfo) <-chan minio.ObjectInfo {
	objCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objCh)
		for _, object := range objects {
			objCh <- object
		}
	}()

	return objCh
}

func TestHandleBucketView(t *testing.T) {
	t.Parallel()

	// Two versions of the same object, as a version-aware listing returns them.
	listVersions := func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
		if !opts.WithVersions {
			return objectChan()
		}
		return objectChan(
			minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v2-abcdefghijk", IsLatest: true},
			minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk"},
		)
	}

	cases := []struct {
		it                   string
		instanceName         string
		listObjectsFunc      func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo
		path                 string
		rootURL              string
		showVersions         bool
		showMetadata         bool
		expectedStatusCode   int
		expectedBodyContains []string
		unexpectedInBody     []string
	}{
		{
			it: "renders a bucket containing a file",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			expectedStatusCode: http.StatusOK,
			// The listing defaults to sorting by key, ascending.
			expectedBodyContains: []string{"FILE-NAME", "arrow_upward"},
		},
		{
			it: "renders placeholder for an empty bucket",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan()
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"No objects in"},
		},
		{
			it: "renders a bucket containing an archive",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "archive.tar.gz"})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"archive"},
		},
		{
			it: "renders a bucket containing an image",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME.png"})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"photo"},
		},
		{
			it: "renders a bucket containing a sound file",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME.mp3"})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"music_note"},
		},
		{
			it: "renders a bucket with a folder",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "AFolder/"})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"folder"},
		},
		{
			it: "renders the path inside the bucket",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan()
			},
			path:                 "abc/def",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"def"},
		},
		{
			it: "prefixes all links with the root URL",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan()
			},
			path:               "abc/def",
			rootURL:            "/rootTest",
			expectedStatusCode: http.StatusOK,
			expectedBodyContains: []string{
				`<a class="link" href="/rootTest/primary/buckets/BUCKET-NAME/">BUCKET-NAME</a>`,
			},
		},
		{
			it:           "returns not found for an unknown instance",
			instanceName: "unknown",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan()
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: []string{"Instance not found"},
		},
		{
			it: "shows an error message if the bucket doesn't exist",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Err: errBucketDoesNotExist})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"does not exist on S3 instance"},
		},
		{
			it: "shows an error message if there is an S3 error",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Err: errS3})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"Unable to list objects", errS3.Error()},
		},
		{
			it:                   "does not show version columns when ShowVersions is disabled",
			listObjectsFunc:      listVersions,
			showVersions:         false,
			expectedStatusCode:   http.StatusOK,
			unexpectedInBody:     []string{"Version ID", "v1-abcdef", "v2-abcdef"},
			expectedBodyContains: []string{"No objects in"},
		},
		{
			it:                 "renders multiple versions when ShowVersions is enabled",
			listObjectsFunc:    listVersions,
			showVersions:       true,
			expectedStatusCode: http.StatusOK,
			expectedBodyContains: []string{
				"Version ID",
				"v1-abcdef",
				"v2-abcdef",
				"Latest",
				// Older versions are collapsed behind a toggle.
				`class="version-row" style="display: none;`,
			},
		},
		{
			it: "falls back to a normal listing when the versioned listing fails",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				if opts.WithVersions {
					return objectChan(minio.ObjectInfo{Err: errS3})
				}
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"FILE-NAME", "Object versions unavailable"},
			unexpectedInBody:     []string{"Version ID"},
		},
		{
			it: "falls back to a normal listing when the versioned listing returns nothing",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				// Some S3-compatible providers silently return an empty result
				// instead of erroring when versioned listing isn't supported.
				if opts.WithVersions {
					return objectChan()
				}
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"FILE-NAME", "Object versions unavailable"},
			unexpectedInBody:     []string{"Version ID"},
		},
		{
			it: "does not warn about unavailable versions for an empty bucket",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan()
			},
			showVersions:       true,
			expectedStatusCode: http.StatusOK,
			unexpectedInBody:   []string{"Object versions unavailable"},
		},
		{
			it: "does not hide folders or objects when the provider never sets IsLatest",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				if !opts.WithVersions {
					return objectChan()
				}
				// Folders synthesized from CommonPrefixes never carry version
				// metadata, and some providers don't reliably set IsLatest on
				// real objects either.
				return objectChan(
					minio.ObjectInfo{Key: "AFolder/"},
					minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk"},
				)
			},
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"AFolder", "FILE-NAME"},
			unexpectedInBody:     []string{`class="version-row" style="display: none;`},
		},
		{
			it: "shows the metadata action when ShowMetadata is enabled",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			showMetadata:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{`onclick="handleOpenMetadataModal(`},
		},
		{
			it: "hides the metadata action when ShowMetadata is disabled",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			showMetadata:         false,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"FILE-NAME"},
			unexpectedInBody:     []string{`onclick="handleOpenMetadataModal(`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListObjectsFunc: tc.listObjectsFunc,
				EndpointURLFunc: mustParseURLFunc("http://localhost:9000"),
			}
			instances := s3manager.S3Instances{{ID: "1", Name: "primary", Client: s3}}
			templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))

			instanceName := tc.instanceName
			if instanceName == "" {
				instanceName = "primary"
			}

			r := mux.NewRouter()
			r.PathPrefix("/{instance}/buckets/").Handler(s3manager.HandleBucketView(instances, templates, s3manager.Options{
				RootURL:       tc.rootURL,
				AllowDelete:   true,
				ListRecursive: true,
				ShowVersions:  tc.showVersions,
				ShowMetadata:  tc.showMetadata,
			})).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(fmt.Sprintf("%s/%s/buckets/BUCKET-NAME/%s", ts.URL, instanceName, tc.path))
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode) // status code
			for _, expected := range tc.expectedBodyContains {
				is.True(strings.Contains(string(body), expected)) // expected body content
			}
			for _, unexpected := range tc.unexpectedInBody {
				is.True(!strings.Contains(string(body), unexpected)) // unexpected body content
			}

			if resp.StatusCode == http.StatusOK {
				backLink := fmt.Sprintf("<a href=%q class=\"button circle transparent\">", tc.rootURL+"/primary/buckets")
				is.True(strings.Contains(string(body), backLink)) // back link honours the root URL
			}
		})
	}
}
