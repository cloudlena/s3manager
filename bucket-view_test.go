package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestBucketViewHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		s3                 S3Client
		bucketName         string
		expectedStatusCode int
		expectedBody       string
	}{
		"bucket doesn't exist": {
			s3:                 &S3ClientMock{},
			bucketName:         "testBucket",
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       "bucket not found\n",
		},
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("internal S3 error"),
			},
			bucketName:         "testBucket",
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "error listing objects\n",
		},
	}

	for _, tc := range tests {
		r := mux.NewRouter()
		r.
			Methods("GET").
			Path("/buckets/{bucketName}").
			Handler(BucketViewHandler(tc.s3))

		ts := httptest.NewServer(r)
		defer ts.Close()

		url := fmt.Sprintf("%s/buckets/%s", ts.URL, tc.bucketName)
		resp, err := http.Get(url)
		assert.NoError(err)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(err)

		assert.Equal(tc.expectedStatusCode, resp.StatusCode)
		assert.Equal(tc.expectedBody, string(body))
	}
}
