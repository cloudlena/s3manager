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
	"github.com/matryer/is"
	"github.com/matryer/way"
	"github.com/minio/minio-go/v7"
)

func TestHandleGetObject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		getObjectFunc        func(context.Context, string, string, minio.GetObjectOptions) (*minio.Object, error)
		bucketName           string
		objectName           string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "returns error if there is an S3 error",
			getObjectFunc: func(context.Context, string, string, minio.GetObjectOptions) (*minio.Object, error) {
				return nil, errS3
			},
			bucketName:           "testBucket",
			objectName:           "testObject",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{
				GetObjectFunc: tc.getObjectFunc,
			}

			r := way.NewRouter()
			r.Handle(http.MethodGet, "/buckets/:bucketName/objects/:objectName", s3manager.HandleGetObject(s3))

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(fmt.Sprintf("%s/buckets/%s/objects/%s", ts.URL, tc.bucketName, tc.objectName))
			is.NoErr(err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
