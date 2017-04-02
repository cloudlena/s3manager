package buckets_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mastertinner/s3-manager/buckets"
	"github.com/mastertinner/s3-manager/mock"
	"github.com/stretchr/testify/assert"
)

func TestCreateHandler(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		description        string
		s3Client           *mock.S3Client
		body               string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			description:        "success",
			s3Client:           &mock.S3Client{},
			body:               "{\"name\":\"myBucket\"}",
			expectedStatusCode: http.StatusCreated,
			expectedBody:       "{\"name\":\"myBucket\",\"creationDate\":\"0001-01-01T00:00:00Z\"}\n",
		},
		{
			description:        "empty request",
			s3Client:           &mock.S3Client{},
			body:               "",
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedBody:       "error decoding json\n",
		},
		{
			description:        "malformed request",
			s3Client:           &mock.S3Client{},
			body:               "}",
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedBody:       "error decoding json\n",
		},
		{
			description: "s3 error",
			s3Client: &mock.S3Client{
				Err: errors.New("internal S3 error"),
			},
			body:               "{\"name\":\"myBucket\"}",
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "error making bucket\n",
		},
	}

	for _, tc := range tests {
		req, err := http.NewRequest("POST", "/api/buckets", bytes.NewBufferString(tc.body))
		assert.NoError(err)

		rr := httptest.NewRecorder()
		handler := buckets.CreateHandler(tc.s3Client)

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code, tc.description)
		assert.Equal(tc.expectedBody, rr.Body.String(), tc.description)
	}
}
