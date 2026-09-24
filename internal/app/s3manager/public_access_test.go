package s3manager_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

func TestPublicObjectURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it           string
		endpoint     string
		bucketLookup minio.BucketLookupType
		publicURL    string
		bucketName   string
		objectName   string
		expectedURL  string
	}{
		{
			it:          "addresses the bucket in the path for other endpoints on auto lookup",
			endpoint:    "http://localhost:9000",
			bucketName:  "my-bucket",
			objectName:  "my-file.txt",
			expectedURL: "http://localhost:9000/my-bucket/my-file.txt",
		},
		{
			it:          "addresses the bucket in the host name for Amazon endpoints on auto lookup",
			endpoint:    "https://s3.amazonaws.com",
			bucketName:  "my-bucket",
			objectName:  "my-file.txt",
			expectedURL: "https://my-bucket.s3.amazonaws.com/my-file.txt",
		},
		{
			it:          "addresses buckets with dots in the path on auto lookup over SSL",
			endpoint:    "https://s3.amazonaws.com",
			bucketName:  "my.bucket",
			objectName:  "my-file.txt",
			expectedURL: "https://s3.amazonaws.com/my.bucket/my-file.txt",
		},
		{
			it:           "addresses the bucket in the host name on DNS lookup",
			endpoint:     "https://minio.example.com",
			bucketLookup: minio.BucketLookupDNS,
			bucketName:   "my-bucket",
			objectName:   "my-file.txt",
			expectedURL:  "https://my-bucket.minio.example.com/my-file.txt",
		},
		{
			it:           "addresses the bucket in the path on path lookup",
			endpoint:     "https://s3.amazonaws.com",
			bucketLookup: minio.BucketLookupPath,
			bucketName:   "my-bucket",
			objectName:   "my-file.txt",
			expectedURL:  "https://s3.amazonaws.com/my-bucket/my-file.txt",
		},
		{
			it:          "fills in a public URL template",
			endpoint:    "http://localhost:9000",
			publicURL:   "https://files.example.com/{bucket}/{key}",
			bucketName:  "my-bucket",
			objectName:  "dir/my-file.txt",
			expectedURL: "https://files.example.com/my-bucket/dir/my-file.txt",
		},
		{
			it:          "fills in a public URL template without a bucket",
			endpoint:    "http://localhost:9000",
			publicURL:   "https://cdn.example.com/{key}",
			bucketName:  "my-bucket",
			objectName:  "my-file.txt",
			expectedURL: "https://cdn.example.com/my-file.txt",
		},
		{
			it:          "escapes the object key but keeps its slashes",
			endpoint:    "http://localhost:9000",
			bucketName:  "my-bucket",
			objectName:  "my dir/file #1?.txt",
			expectedURL: "http://localhost:9000/my-bucket/my%20dir/file%20%231%3F.txt",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			instance := &s3manager.S3Instance{
				Client:       &mocks.S3Mock{EndpointURLFunc: mustParseURLFunc(tc.endpoint)},
				BucketLookup: tc.bucketLookup,
				PublicURL:    tc.publicURL,
			}

			is.Equal(tc.expectedURL, instance.PublicObjectURL(tc.bucketName, tc.objectName))
		})
	}
}

func TestHandleCheckPublicAccess(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it               string
		s3ResponseStatus int
		expectAccessible bool
		expectStatusCode int
		networkError     bool
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

			var s3ServerURL string
			if !tc.networkError {
				s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					is.Equal(http.MethodHead, r.Method)
					is.Equal("/my-bucket/my%20file.txt", r.URL.EscapedPath())
					w.WriteHeader(tc.s3ResponseStatus)
				}))
				defer s3Server.Close()
				s3ServerURL = s3Server.URL
			} else {
				s3ServerURL = "http://localhost:0"
			}

			s3 := &mocks.S3Mock{EndpointURLFunc: mustParseURLFunc(s3ServerURL)}
			instances := s3manager.S3Instances{{ID: "1", Name: "primary", Client: s3}}

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/public-access", s3manager.HandleCheckPublicAccess(instances))

			req := httptest.NewRequest(http.MethodGet, "/primary/api/buckets/my-bucket/objects/my%20file.txt/public-access", nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			is.Equal(http.StatusOK, rr.Code)

			var response map[string]any
			err := json.Unmarshal(rr.Body.Bytes(), &response)
			is.NoErr(err)

			is.Equal(s3ServerURL+"/my-bucket/my%20file.txt", response["url"])
			is.Equal(tc.expectAccessible, response["accessible"])
			is.Equal(float64(tc.expectStatusCode), response["statusCode"])
		})
	}
}

func TestHandleCheckPublicAccessRejectsInvalidBucketNames(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	instances := s3manager.S3Instances{{ID: "1", Name: "primary", Client: &mocks.S3Mock{}}}

	r := mux.NewRouter()
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/public-access", s3manager.HandleCheckPublicAccess(instances))

	req := httptest.NewRequest(http.MethodGet, "/primary/api/buckets/evil.example.com%23/objects/my-file.txt/public-access", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	is.Equal(http.StatusBadRequest, rr.Code)
}
