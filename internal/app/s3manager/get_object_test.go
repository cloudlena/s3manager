package s3manager_test

import (
	"context"
	"fmt"
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

func TestHandleGetObject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		getObjectFunc        func(context.Context, string, string, minio.GetObjectOptions) (*minio.Object, error)
		bucketName           string
		objectName           string
		queryString          string
		showVersions         bool
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "returns error if there is an S3 error",
			getObjectFunc: func(context.Context, string, string, minio.GetObjectOptions) (*minio.Object, error) {
				return nil, errS3
			},
			bucketName:           "BUCKET-NAME",
			objectName:           "OBJECT-NAME",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
		{
			it: "leaves VersionID empty when no versionId query param is given",
			getObjectFunc: func(_ context.Context, _, _ string, opts minio.GetObjectOptions) (*minio.Object, error) {
				if opts.VersionID != "" {
					return nil, fmt.Errorf("expected empty VersionID, got %q", opts.VersionID)
				}
				return nil, errS3
			},
			bucketName:           "BUCKET-NAME",
			objectName:           "OBJECT-NAME",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
		{
			it: "passes the versionId query param through to GetObjectOptions",
			getObjectFunc: func(_ context.Context, _, _ string, opts minio.GetObjectOptions) (*minio.Object, error) {
				if opts.VersionID != "VERSION-123" {
					return nil, fmt.Errorf("expected VersionID %q, got %q", "VERSION-123", opts.VersionID)
				}
				return nil, errS3
			},
			bucketName:           "BUCKET-NAME",
			objectName:           "OBJECT-NAME",
			queryString:          "?versionId=VERSION-123",
			showVersions:         true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
		{
			it: "ignores the versionId query param when showVersions is disabled",
			getObjectFunc: func(_ context.Context, _, _ string, opts minio.GetObjectOptions) (*minio.Object, error) {
				if opts.VersionID != "" {
					return nil, fmt.Errorf("expected empty VersionID, got %q", opts.VersionID)
				}
				return nil, errS3
			},
			bucketName:           "BUCKET-NAME",
			objectName:           "OBJECT-NAME",
			queryString:          "?versionId=VERSION-123",
			showVersions:         false,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "mocked s3 error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				GetObjectFunc: tc.getObjectFunc,
			}

			r := mux.NewRouter()
			r.Handle("/buckets/{bucketName}/objects/{objectName}", s3manager.HandleGetObject(s3, true, tc.showVersions)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(fmt.Sprintf("%s/buckets/%s/objects/%s%s", ts.URL, tc.bucketName, tc.objectName, tc.queryString))
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
