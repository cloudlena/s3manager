package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateBucketHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		s3                 S3Client
		body               string
		expectedStatusCode int
		expectedBody       string
	}{
		"success": {
			s3:                 &S3ClientMock{},
			body:               "{\"name\":\"myBucket\"}",
			expectedStatusCode: http.StatusCreated,
			expectedBody:       "{\"name\":\"myBucket\",\"creationDate\":\"0001-01-01T00:00:00Z\"}\n",
		},
		"empty request": {
			s3:                 &S3ClientMock{},
			body:               "",
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedBody:       "error decoding json\n",
		},
		"malformed request": {
			s3:                 &S3ClientMock{},
			body:               "}",
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedBody:       "error decoding json\n",
		},
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("mocked S3 error"),
			},
			body:               "{\"name\":\"myBucket\"}",
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "error making bucket\n",
		},
	}

	for tcID, tc := range tests {
		req, err := http.NewRequest(http.MethodPost, "/api/buckets", bytes.NewBufferString(tc.body))
		assert.NoError(err, tcID)

		rr := httptest.NewRecorder()
		handler := CreateBucketHandler(tc.s3)

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code, tcID)
		assert.Equal(tc.expectedBody, rr.Body.String(), tcID)
	}
}
