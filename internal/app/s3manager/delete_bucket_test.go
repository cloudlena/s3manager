package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/stretchr/testify/assert"
)

func TestDeleteBucketHandler(t *testing.T) {
	cases := map[string]struct {
		removeBucketFunc     func(string) error
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"deletes an existing bucket": {
			removeBucketFunc: func(bucketName string) error {
				return nil
			},
			expectedStatusCode:   http.StatusNoContent,
			expectedBodyContains: "",
		},
		"returns error if there is an S3 error": {
			removeBucketFunc: func(bucketName string) error {
				return errors.New("mocked S3 error")
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			assert := assert.New(t)

			s3 := &S3Mock{
				RemoveBucketFunc: tc.removeBucketFunc,
			}

			req, err := http.NewRequest(http.MethodDelete, "/api/buckets/bucketName", nil)
			assert.NoError(err, tcID)

			rr := httptest.NewRecorder()
			handler := s3manager.DeleteBucketHandler(s3)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()

			assert.Equal(tc.expectedStatusCode, resp.StatusCode, tcID)
			assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
		})
	}
}
