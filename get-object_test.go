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

func TestGetObjectHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		s3                    S3Client
		bucketName            string
		objectName            string
		expectedStatusCode    int
		expectedBodyCountains string
	}{
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("mocked S3 error"),
			},
			bucketName:            "testBucket",
			objectName:            "testObject",
			expectedStatusCode:    http.StatusInternalServerError,
			expectedBodyCountains: "error getting object\n",
		},
	}

	for _, tc := range tests {
		r := mux.NewRouter()
		r.
			Methods(http.MethodGet).
			Path("/buckets/{bucketName}/objects/{objectName}").
			Handler(GetObjectHandler(tc.s3))

		ts := httptest.NewServer(r)
		defer ts.Close()

		url := fmt.Sprintf("%s/buckets/%s/objects/%s", ts.URL, tc.bucketName, tc.objectName)
		resp, err := http.Get(url)
		assert.NoError(err)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(err)

		assert.Equal(tc.expectedStatusCode, resp.StatusCode)
		assert.Contains(string(body), tc.expectedBodyCountains)
	}
}
