package s3manager_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
)

func TestHandleCheckPublicAccess(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                 string
		s3ResponseStatus   int
		expectAccessible   bool
		expectStatusCode   int
		networkError       bool
	}{
		{
			it:               "reports accessible when S3 returns 200 OK",
			s3ResponseStatus: http.StatusOK,
			expectAccessible: true,
			expectStatusCode: http.StatusOK,
		},
		{
			it:               "reports not accessible when S3 returns 403 Forbidden",
			s3ResponseStatus: http.StatusForbidden,
			expectAccessible: false,
			expectStatusCode: http.StatusForbidden,
		},
		{
			it:               "reports not accessible when S3 returns 404 Not Found",
			s3ResponseStatus: http.StatusNotFound,
			expectAccessible: false,
			expectStatusCode: http.StatusNotFound,
		},
		{
			it:               "reports not accessible on network error",
			networkError:     true,
			expectAccessible: false,
			expectStatusCode: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			is := is.New(t)

			// Start a mock S3 server to respond to the HEAD request
			var s3ServerURL string
			if !tc.networkError {
				s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					is.Equal(http.MethodHead, r.Method)
					w.WriteHeader(tc.s3ResponseStatus)
				}))
				defer s3Server.Close()
				s3ServerURL = s3Server.URL
			} else {
				// Use an invalid port to simulate a network error
				s3ServerURL = "http://localhost:0"
			}

			s3 := &mocks.S3Mock{
				EndpointURLFunc: func() *url.URL {
					u, _ := url.Parse(s3ServerURL)
					return u
				},
			}

			handler := s3manager.HandleCheckPublicAccess(s3)
			r := mux.NewRouter()
			r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}/public-access", handler)

			req := httptest.NewRequest(http.MethodGet, "/api/buckets/my-bucket/objects/my-file.txt/public-access", nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			is.Equal(http.StatusOK, rr.Code)

			var response map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			is.NoErr(err)

			is.Equal(tc.expectAccessible, response["accessible"])
			is.Equal(float64(tc.expectStatusCode), response["statusCode"])
		})
	}
}
