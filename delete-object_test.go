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
		s3                   S3Client
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"success": {
			s3:                   &S3ClientMock{},
			expectedStatusCode:   http.StatusNoContent,
			expectedBodyContains: "",
		},
		"s3 error": {
			s3: &S3ClientMock{
				Err: errors.New("mocked S3 error"),
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range tests {
		req, err := http.NewRequest(http.MethodDelete, "/api/buckets/bucketName/objects/objectName", nil)
		assert.NoError(err, tcID)

		rr := httptest.NewRecorder()
		handler := DeleteObjectHandler(tc.s3)

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code, tcID)
		assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
	}
}
