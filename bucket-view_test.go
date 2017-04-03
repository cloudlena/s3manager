package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestBucketViewHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		s3                    S3Client
		bucketName            string
		expectedStatusCode    int
		expectedBodyCountains string
	}{
		"success (empty bucket)": {
			s3: &S3ClientMock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
			},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusOK,
			expectedBodyCountains: "No objects in",
		},
		"success (with file)": {
			s3: &S3ClientMock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
				Objects: []minio.ObjectInfo{
					{Key: "testFile"},
				},
			},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusOK,
			expectedBodyCountains: "testBucket",
		},
		"success (archive)": {
			s3: &S3ClientMock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
				Objects: []minio.ObjectInfo{
					{Key: "archive.tar.gz"},
				},
			},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusOK,
			expectedBodyCountains: "archive",
		},
		"success (image)": {
			s3: &S3ClientMock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
				Objects: []minio.ObjectInfo{
					{Key: "testImage.png"},
				},
			},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusOK,
			expectedBodyCountains: "photo",
		},
		"success (sound)": {
			s3: &S3ClientMock{
				Buckets: []minio.BucketInfo{
					{Name: "testBucket"},
				},
				Objects: []minio.ObjectInfo{
					{Key: "testSound.mp3"},
				},
			},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusOK,
			expectedBodyCountains: "music_note",
		},
		"bucket doesn't exist": {
			s3:                    &S3ClientMock{},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusNotFound,
			expectedBodyCountains: "bucket not found\n",
		},
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("mocked S3 error"),
			},
			bucketName:            "testBucket",
			expectedStatusCode:    http.StatusInternalServerError,
			expectedBodyCountains: "error listing objects\n",
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
		assert.Contains(string(body), tc.expectedBodyCountains)
	}
}
