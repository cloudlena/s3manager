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
	"sync"
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
			it: "names the region of a bucket in another region",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Err: minio.ErrorResponse{Code: "PermanentRedirect", Region: "us-west-2"}})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"is located in region", "us-west-2", "Please set the REGION"},
		},
		{
			it: "asks for the region of a bucket in an unknown other region",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Err: minio.ErrorResponse{Code: "PermanentRedirect"}})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"is located in another region", "set the instance"},
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
				`class="version-row `,
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
			it: "falls back to a V1 listing when the V2 listing returns nothing",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				// Some S3-compatible providers answer a ListObjects V2 request
				// they don't implement with an empty result instead of erroring.
				if !opts.UseV1 {
					return objectChan()
				}
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"FILE-NAME"},
			unexpectedInBody:     []string{"No objects in"},
		},
		{
			it: "falls back to a V1 listing in a full listing too",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				if opts.WithVersions || !opts.UseV1 {
					return objectChan()
				}
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			// Showing versions takes the full-listing path instead of the
			// cursor-paged one.
			showVersions:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"FILE-NAME"},
			unexpectedInBody:     []string{"No objects in"},
		},
		{
			it: "keeps the empty listing when the V1 fallback fails",
			listObjectsFunc: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				if !opts.UseV1 {
					return objectChan()
				}
				return objectChan(minio.ObjectInfo{Err: errS3})
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"No objects in"},
			unexpectedInBody:     []string{"Unable to list objects"},
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
			unexpectedInBody:     []string{`class="version-row `},
		},
		{
			it: "shows the metadata action when ShowMetadata is enabled",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			showMetadata:         true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{`onclick="openMetadataDialog(`},
		},
		{
			it: "hides the metadata action when ShowMetadata is disabled",
			listObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				return objectChan(minio.ObjectInfo{Key: "FILE-NAME"})
			},
			showMetadata:         false,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: []string{"FILE-NAME"},
			unexpectedInBody:     []string{`onclick="openMetadataDialog(`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListObjectsFunc: tc.listObjectsFunc,
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

// objectStream serves objects on a ListObjects channel and gives up as soon as
// the context is cancelled, the way minio's own listing does when the consumer
// stops reading.
func objectStream(ctx context.Context, objects []minio.ObjectInfo) <-chan minio.ObjectInfo {
	objCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objCh)
		for _, object := range objects {
			select {
			case <-ctx.Done():
				return
			case objCh <- object:
			}
		}
	}()

	return objCh
}

// fakeListing serves a bucket listing of fixed keys, honouring StartAfter the
// way S3 does, and records the options every call was made with.
type fakeListing struct {
	mu    sync.Mutex
	calls []minio.ListObjectsOptions
}

// listFunc returns a ListObjects implementation serving the given keys. They are
// served in the order given, which is not necessarily key order: a delimited S3
// listing reports a batch's objects before its folders.
func (f *fakeListing) listFunc(keys ...string) func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return func(ctx context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
		f.mu.Lock()
		f.calls = append(f.calls, opts)
		f.mu.Unlock()

		var objects []minio.ObjectInfo
		for _, key := range keys {
			if key > opts.StartAfter {
				objects = append(objects, minio.ObjectInfo{Key: key})
			}
		}

		return objectStream(ctx, objects)
	}
}

// only returns the single set of options the listing was asked for, failing if
// it was called any other number of times.
func (f *fakeListing) only(t *testing.T) minio.ListObjectsOptions {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.calls) != 1 {
		t.Fatalf("expected exactly one listing, got %d", len(f.calls))
	}

	return f.calls[0]
}

// paddedKeys returns count keys that sort in the order they are generated.
func paddedKeys(count int) []string {
	keys := make([]string, 0, count)
	for i := range count {
		keys = append(keys, fmt.Sprintf("object-%05d", i))
	}

	return keys
}

// getBucketView renders the bucket view of a bucket served by the given listing
// and returns the response body.
func getBucketView(t *testing.T, listObjects func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo, query url.Values) string {
	t.Helper()
	is := is.New(t)

	s3 := &mocks.S3Mock{
		ListObjectsFunc: listObjects,
	}
	instances := s3manager.S3Instances{{ID: "1", Name: "primary", Client: s3}}
	templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))

	r := mux.NewRouter()
	r.PathPrefix("/{instance}/buckets/").Handler(s3manager.HandleBucketView(instances, templates, s3manager.Options{
		AllowDelete: true,
	})).Methods(http.MethodGet)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/primary/buckets/BUCKET-NAME/?" + query.Encode())
	is.NoErr(err)
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()
	body, err := io.ReadAll(resp.Body)
	is.NoErr(err)
	is.Equal(http.StatusOK, resp.StatusCode) // status code

	return string(body)
}

func TestHandleBucketViewPaging(t *testing.T) {
	t.Parallel()

	t.Run("asks S3 for a single page of the default view", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		listing := &fakeListing{}
		body := getBucketView(t, listing.listFunc(paddedKeys(60)...), nil)

		opts := listing.only(t)
		is.Equal("", opts.StartAfter)                                   // first page starts at the beginning
		is.Equal(26, opts.MaxKeys)                                      // one object beyond the page, to detect a next one
		is.True(strings.Contains(body, "Showing 1–25"))                 // page range without a total
		is.True(!strings.Contains(body, "of 60"))                       // the total is unknown
		is.True(strings.Contains(body, "object-00024"))                 // last object of the page
		is.True(!strings.Contains(body, "object-00025"))                // first object of the next page
		is.True(strings.Contains(body, `goToNextPage('object-00024')`)) // next page resumes after it
	})

	t.Run("resumes after the cursor trail", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		listing := &fakeListing{}
		body := getBucketView(t, listing.listFunc(paddedKeys(60)...), url.Values{
			"cursor": {"object-00024", "object-00049"},
		})

		is.Equal("object-00049", listing.only(t).StartAfter)                         // resumes after the last cursor
		is.True(strings.Contains(body, "Showing 51–60"))                             // the trail gives an exact offset
		is.True(strings.Contains(body, `<button class="circle primary">3</button>`)) // and an exact page number
		is.True(strings.Contains(body, "object-00050"))                              // first object of the page
		is.True(!strings.Contains(body, "object-00049"))                             // last object of the previous page
	})

	t.Run("steps past the contents of a folder", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		keys := []string{"dir/", "dir/nested-object", "zebra"}

		listing := &fakeListing{}
		body := getBucketView(t, listing.listFunc(keys...), url.Values{"perPage": {"1"}})

		// Resuming after the folder's own key would collapse its contents into
		// the same folder entry again, listing it forever.
		endOfDir := "dir/\U0010FFFF"
		is.True(strings.Contains(body, "goToNextPage('dir\\/\U0010FFFF')")) // cursor clears the folder

		listing = &fakeListing{}
		body = getBucketView(t, listing.listFunc(keys...), url.Values{
			"perPage": {"1"},
			"cursor":  {endOfDir},
		})

		is.True(strings.Contains(body, "zebra"))          // the entry after the folder
		is.True(!strings.Contains(body, "nested-object")) // which sits inside the folder, not next to it
	})

	t.Run("puts a page in key order when a listing reports folders last", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		listing := &fakeListing{}
		// A delimited listing reports a batch's objects before its folders, so
		// the batch arrives out of key order.
		body := getBucketView(t, listing.listFunc("b-object", "a-folder/"), nil)

		is.True(strings.Index(body, "a-folder") < strings.Index(body, "b-object")) // shown in key order
	})

	t.Run("falls back to a full listing when it cannot page in S3", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			it    string
			query url.Values
		}{
			{it: "sorting by another column", query: url.Values{"sortBy": {"size"}}},
			{it: "sorting descending", query: url.Values{"sortOrder": {"desc"}}},
			{it: "searching", query: url.Values{"search": {"object-0001"}}},
			{it: "showing every object", query: url.Values{"perPage": {"0"}}},
		}

		for _, tc := range cases {
			t.Run(tc.it, func(t *testing.T) {
				t.Parallel()
				is := is.New(t)

				listing := &fakeListing{}
				body := getBucketView(t, listing.listFunc(paddedKeys(60)...), tc.query)

				opts := listing.only(t)
				is.Equal("", opts.StartAfter) // a full listing starts at the beginning
				is.Equal(0, opts.MaxKeys)     // and is not cut into pages
				is.True(strings.Contains(body, " of "))
			})
		}
	})

	t.Run("caps a full listing and says that it is partial", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		listing := &fakeListing{}
		body := getBucketView(t, listing.listFunc(paddedKeys(10_001)...), url.Values{"sortOrder": {"desc"}})

		is.True(strings.Contains(body, "Partial listing")) // the listing hit the cap
		is.True(strings.Contains(body, "of 10000"))        // and counts only what it listed
	})

	t.Run("does not call a capped listing partial", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		listing := &fakeListing{}
		body := getBucketView(t, listing.listFunc(paddedKeys(10_000)...), url.Values{"sortOrder": {"desc"}})

		is.True(!strings.Contains(body, "Partial listing")) // exactly at the cap is still complete
		is.True(strings.Contains(body, "of 10000"))
	})
}
