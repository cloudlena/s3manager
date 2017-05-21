package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mastertinner/s3manager"
	"github.com/stretchr/testify/assert"
)

func TestDeleteBucketHandler(t *testing.T) {
	assert := assert.New(t)

	cases := map[string]struct {
		s3                   s3manager.S3
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"success": {
			s3:                   &s3Mock{},
			expectedStatusCode:   http.StatusNoContent,
			expectedBodyContains: "",
		},
		"s3 error": {
			s3: &s3Mock{
				Err: errors.New("mocked S3 error"),
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		req, err := http.NewRequest(http.MethodDelete, "/api/buckets/bucketName", nil)
		assert.NoError(err, tcID)

		rr := httptest.NewRecorder()
		handler := s3manager.DeleteBucketHandler(tc.s3)

		handler.ServeHTTP(rr, req)
		resp := rr.Result()

		assert.Equal(tc.expectedStatusCode, resp.StatusCode, tcID)
		assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
	}
}
