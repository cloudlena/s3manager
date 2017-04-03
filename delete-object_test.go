package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteObjectHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		s3                 S3Client
		expectedStatusCode int
		expectedBody       string
	}{
		"success": {
			s3:                 &S3ClientMock{},
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       "",
		},
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("internal S3 error"),
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "error removing object\n",
		},
	}

	for _, tc := range tests {
		req, err := http.NewRequest("DELETE", "/api/buckets/bucketName/objects/objectName", nil)
		assert.NoError(err)

		rr := httptest.NewRecorder()
		handler := DeleteObjectHandler(tc.s3)

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code)
		assert.Equal(tc.expectedBody, rr.Body.String())
	}
}
