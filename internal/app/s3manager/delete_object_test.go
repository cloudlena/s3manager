package s3manager_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/stretchr/testify/assert"
)

func TestDeleteObjectHandler(t *testing.T) {
	cases := map[string]struct {
		s3                   s3manager.S3
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"deletes an existing object": {
			s3:                   &s3Mock{},
			expectedStatusCode:   http.StatusNoContent,
			expectedBodyContains: "",
		},
		"returns error if there is an S3 error": {
			s3: &s3Mock{
				Err: errors.New("mocked S3 error"),
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			assert := assert.New(t)

			req, err := http.NewRequest(http.MethodDelete, "/api/buckets/bucketName/objects/objectName", nil)
			assert.NoError(err, tcID)

			rr := httptest.NewRecorder()
			handler := s3manager.DeleteObjectHandler(tc.s3)

			handler.ServeHTTP(rr, req)

			assert.Equal(tc.expectedStatusCode, rr.Code, tcID)
			assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
		})
	}
}
