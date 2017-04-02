package buckets_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mastertinner/s3-manager/buckets"
	"github.com/mastertinner/s3-manager/mock"
	"github.com/stretchr/testify/assert"
)

func TestDeleteHandler(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		description        string
		s3Client           *mock.S3Client
		expectedStatusCode int
		expectedBody       string
	}{
		{
			description:        "success",
			s3Client:           &mock.S3Client{},
			expectedStatusCode: http.StatusNoContent,
			expectedBody:       "",
		},
		{
			description: "s3 error",
			s3Client: &mock.S3Client{
				Err: errors.New("internal S3 error"),
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "error removing bucket\n",
		},
	}

	for _, tc := range tests {
		req, err := http.NewRequest("DELETE", "/api/buckets/bucketName", nil)
		assert.NoError(err)

		rr := httptest.NewRecorder()
		handler := buckets.DeleteHandler(tc.s3Client)

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code, tc.description)
		assert.Equal(tc.expectedBody, rr.Body.String(), tc.description)
	}
}
