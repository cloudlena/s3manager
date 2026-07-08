package s3manager_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestHandleGetObjectMetadata(t *testing.T) {
	t.Parallel()

	lastModified := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)

	cases := []struct {
		it                 string
		statObjectFunc     func(context.Context, string, string, minio.StatObjectOptions) (minio.ObjectInfo, error)
		queryString        string
		expectedStatusCode int
		expectedBody       map[string]any
		expectedBodyError  string
	}{
		{
			it: "returns metadata for the latest version",
			statObjectFunc: func(_ context.Context, _, _ string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
				is := is.New(t)
				is.Equal("", opts.VersionID)
				return minio.ObjectInfo{
					Key:          "OBJECT-NAME",
					Size:         1234,
					ContentType:  "text/plain",
					ETag:         "abc123",
					LastModified: lastModified,
					StorageClass: "STANDARD",
					UserMetadata: map[string]string{"foo": "bar"},
				}, nil
			},
			expectedStatusCode: http.StatusOK,
			expectedBody: map[string]any{
				"key":          "OBJECT-NAME",
				"size":         float64(1234),
				"contentType":  "text/plain",
				"etag":         "abc123",
				"storageClass": "STANDARD",
				"userMetadata": map[string]any{"foo": "bar"},
			},
		},
		{
			it: "passes the versionId query param through to StatObjectOptions",
			statObjectFunc: func(_ context.Context, _, _ string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
				is := is.New(t)
				is.Equal("VERSION-123", opts.VersionID)
				return minio.ObjectInfo{
					Key:          "OBJECT-NAME",
					VersionID:    "VERSION-123",
					IsLatest:     true,
					LastModified: lastModified,
				}, nil
			},
			queryString:        "?versionId=VERSION-123",
			expectedStatusCode: http.StatusOK,
			expectedBody: map[string]any{
				"key":       "OBJECT-NAME",
				"versionId": "VERSION-123",
				"isLatest":  true,
			},
		},
		{
			it: "derives user metadata from raw headers when UserMetadata is empty",
			statObjectFunc: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
				return minio.ObjectInfo{
					Key:          "OBJECT-NAME",
					LastModified: lastModified,
					Metadata: map[string][]string{
						"X-Amz-Meta-Foo": {"bar"},
						"Content-Type":   {"text/plain"},
					},
				}, nil
			},
			expectedStatusCode: http.StatusOK,
			expectedBody: map[string]any{
				"userMetadata": map[string]any{"foo": "bar"},
			},
		},
		{
			it: "returns error if there is an S3 error",
			statObjectFunc: func(context.Context, string, string, minio.StatObjectOptions) (minio.ObjectInfo, error) {
				return minio.ObjectInfo{}, errS3
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBodyError:  "mocked s3 error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				StatObjectFunc: tc.statObjectFunc,
			}

			r := mux.NewRouter()
			r.Handle("/api/buckets/{bucketName}/objects/{objectName}/metadata", s3manager.HandleGetObjectMetadata(s3)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/api/buckets/BUCKET-NAME/objects/OBJECT-NAME/metadata" + tc.queryString)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()

			is.Equal(tc.expectedStatusCode, resp.StatusCode)

			if tc.expectedBodyError != "" {
				rawBody, err := io.ReadAll(resp.Body)
				is.NoErr(err)
				is.True(strings.Contains(string(rawBody), tc.expectedBodyError))
				return
			}

			var body map[string]any
			is.NoErr(json.NewDecoder(resp.Body).Decode(&body))
			for key, expected := range tc.expectedBody {
				is.Equal(expected, body[key])
			}
		})
	}
}
