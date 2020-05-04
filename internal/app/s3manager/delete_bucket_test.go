package s3manager_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/mastertinner/s3manager/internal/app/s3manager/mocks"
	"github.com/matryer/is"
)

func TestHandleDeleteBucket(t *testing.T) {
	cases := []struct {
		it                   string
		removeBucketFunc     func(string) error
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "deletes an existing bucket",
			removeBucketFunc: func(string) error {
				return nil
			},
			expectedStatusCode:   http.StatusNoContent,
			expectedBodyContains: "",
		},
		{
			it: "returns error if there is an S3 error",
			removeBucketFunc: func(string) error {
				return errS3
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			is := is.New(t)

			s3 := &mocks.S3Mock{
				RemoveBucketFunc: tc.removeBucketFunc,
			}

			req, err := http.NewRequest(http.MethodDelete, "/api/buckets/bucketName", nil)
			is.NoErr(err)

			rr := httptest.NewRecorder()
			handler := s3manager.HandleDeleteBucket(s3)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
