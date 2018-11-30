package s3manager_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/mastertinner/s3manager/internal/app/s3manager/mocks"
	"github.com/matryer/is"
	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
)

func TestHandleGetObject(t *testing.T) {
	cases := map[string]struct {
		getObjectFunc        func(string, string, minio.GetObjectOptions) (*minio.Object, error)
		bucketName           string
		objectName           string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"returns error if there is an S3 error": {
			getObjectFunc: func(string, string, minio.GetObjectOptions) (*minio.Object, error) {
				return nil, errors.New("mocked S3 error")
			},
			bucketName:           "testBucket",
			objectName:           "testObject",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			is := is.New(t)

			s3 := &mocks.S3Mock{
				GetObjectFunc: tc.getObjectFunc,
			}

			r := way.NewRouter()
			r.Handle(http.MethodGet, "/buckets/:bucketName/objects/:objectName", s3manager.HandleGetObject(s3))

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := fmt.Sprintf("%s/buckets/%s/objects/%s", ts.URL, tc.bucketName, tc.objectName)
			resp, err := http.Get(url)
			is.NoErr(err)
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
