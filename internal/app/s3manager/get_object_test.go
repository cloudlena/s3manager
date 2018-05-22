package s3manager_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mastertinner/s3manager/internal/app/s3manager"
	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestGetObjectHandler(t *testing.T) {
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
			assert := assert.New(t)

			s3 := &S3Mock{
				GetObjectFunc: tc.getObjectFunc,
			}

			r := mux.NewRouter()
			r.
				Methods(http.MethodGet).
				Path("/buckets/{bucketName}/objects/{objectName}").
				Handler(s3manager.GetObjectHandler(s3))

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := fmt.Sprintf("%s/buckets/%s/objects/%s", ts.URL, tc.bucketName, tc.objectName)
			resp, err := http.Get(url)
			assert.NoError(err)
			defer func() {
				err = resp.Body.Close()
				if err != nil {
					t.FailNow()
				}
			}()

			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(err)

			assert.Equal(tc.expectedStatusCode, resp.StatusCode)
			assert.Contains(string(body), tc.expectedBodyContains)
		})
	}
}
