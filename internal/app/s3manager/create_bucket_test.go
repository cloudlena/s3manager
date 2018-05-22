package s3manager_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/stretchr/testify/assert"
)

func TestCreateBucketHandler(t *testing.T) {
	cases := map[string]struct {
		makeBucketFunc       func(string, string) error
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"creates a new bucket": {
			makeBucketFunc: func(string, string) error {
				return nil
			},
			body:                 `{"name":"myBucket"}`,
			expectedStatusCode:   http.StatusCreated,
			expectedBodyContains: `{"name":"myBucket","creationDate":"0001-01-01T00:00:00Z"}`,
		},
		"returns error for empty request": {
			makeBucketFunc: func(string, string) error {
				return nil
			},
			body:                 "",
			expectedStatusCode:   http.StatusUnprocessableEntity,
			expectedBodyContains: http.StatusText(http.StatusUnprocessableEntity),
		},
		"returns error for malformed request": {
			makeBucketFunc: func(string, string) error {
				return nil
			},
			body:                 "}",
			expectedStatusCode:   http.StatusUnprocessableEntity,
			expectedBodyContains: http.StatusText(http.StatusUnprocessableEntity),
		},
		"returns error if there is an S3 error": {
			makeBucketFunc: func(string, string) error {
				return errors.New("mocked S3 error")
			},
			body:                 `{"name":"myBucket"}`,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			assert := assert.New(t)

			s3 := &S3Mock{
				MakeBucketFunc: tc.makeBucketFunc,
			}

			req, err := http.NewRequest(http.MethodPost, "/api/buckets", bytes.NewBufferString(tc.body))
			assert.NoError(err)

			rr := httptest.NewRecorder()
			handler := s3manager.CreateBucketHandler(s3)

			handler.ServeHTTP(rr, req)
			resp := rr.Result()

			assert.Equal(tc.expectedStatusCode, resp.StatusCode)
			assert.Contains(rr.Body.String(), tc.expectedBodyContains)
		})
	}
}
