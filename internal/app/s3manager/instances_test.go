package s3manager_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
)

func TestNewS3Instances(t *testing.T) {
	t.Parallel()

	instanceConfig := func(name, signatureType string) s3manager.S3InstanceConfig {
		return s3manager.S3InstanceConfig{
			Name:            name,
			Endpoint:        "localhost:9000",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			SignatureType:   signatureType,
		}
	}

	cases := []struct {
		it          string
		configs     []s3manager.S3InstanceConfig
		expectError bool
	}{
		{
			it:      "creates an instance with V4 signature type",
			configs: []s3manager.S3InstanceConfig{instanceConfig("test", "V4")},
		},
		{
			it:      "creates an instance with V2 signature type",
			configs: []s3manager.S3InstanceConfig{instanceConfig("test", "V2")},
		},
		{
			it:      "creates an instance with V4Streaming signature type",
			configs: []s3manager.S3InstanceConfig{instanceConfig("test", "V4Streaming")},
		},
		{
			it: "creates an instance with Anonymous signature type",
			configs: []s3manager.S3InstanceConfig{
				{Name: "test", Endpoint: "localhost:9000", SignatureType: "Anonymous"},
			},
		},
		{
			it: "creates an instance with IAM credentials",
			configs: []s3manager.S3InstanceConfig{
				{Name: "test", Endpoint: "localhost:9000", UseIam: true},
			},
		},
		{
			it: "creates an instance with a region",
			configs: []s3manager.S3InstanceConfig{
				{
					Name:            "test",
					Endpoint:        "s3.amazonaws.com",
					AccessKeyID:     "key",
					SecretAccessKey: "secret",
					SignatureType:   "V4",
					Region:          "us-east-1",
				},
			},
		},
		{
			it:      "creates multiple instances",
			configs: []s3manager.S3InstanceConfig{instanceConfig("first", "V4"), instanceConfig("second", "V4")},
		},
		{
			it:          "returns an error for empty configs",
			configs:     []s3manager.S3InstanceConfig{},
			expectError: true,
		},
		{
			it: "returns an error for an invalid signature type",
			configs: []s3manager.S3InstanceConfig{
				{Name: "test", Endpoint: "localhost:9000", SignatureType: "INVALID"},
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			instances, err := s3manager.NewS3Instances(tc.configs)
			if tc.expectError {
				is.True(err != nil)
				is.Equal(0, len(instances))
				return
			}

			is.NoErr(err)
			is.Equal(len(tc.configs), len(instances))
			for i, instance := range instances {
				is.Equal(tc.configs[i].Name, instance.Name)
				is.True(instance.Client != nil)
			}
		})
	}
}

func TestS3InstancesGet(t *testing.T) {
	t.Parallel()

	instances := s3manager.S3Instances{
		{ID: "1", Name: "primary"},
		{ID: "2", Name: "secondary"},
	}

	cases := []struct {
		it           string
		identifier   string
		expectedName string
		expectError  bool
	}{
		{it: "finds an instance by ID", identifier: "2", expectedName: "secondary"},
		{it: "finds an instance by name", identifier: "primary", expectedName: "primary"},
		{it: "returns an error for an unknown identifier", identifier: "does-not-exist", expectError: true},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			instance, err := instances.Get(tc.identifier)
			if tc.expectError {
				is.True(err != nil)
				return
			}

			is.NoErr(err)
			is.Equal(tc.expectedName, instance.Name)
		})
	}
}

func TestWithInstance(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:                   "delegates to the handler of the addressed instance",
			instanceName:         "primary",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "http://localhost:9000",
		},
		{
			it:                   "returns not found for an unknown instance",
			instanceName:         "unknown",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3 := &mocks.S3Mock{EndpointURLFunc: mustParseURLFunc("http://localhost:9000")}
			instances := s3manager.S3Instances{{ID: "1", Name: "primary", Client: s3}}

			// The handler writes something only it can know: the endpoint of
			// the client it was given.
			handler := func(s3 s3manager.S3) http.HandlerFunc {
				return func(w http.ResponseWriter, _ *http.Request) {
					_, err := io.WriteString(w, s3.EndpointURL().String())
					is.NoErr(err)
				}
			}

			r := mux.NewRouter()
			r.Handle("/{instance}/endpoint", s3manager.WithInstance(instances, handler)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/endpoint")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}

func TestHandleGetS3Instances(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	instances := s3manager.S3Instances{
		{ID: "1", Name: "first"},
		{ID: "2", Name: "second"},
	}

	req, err := http.NewRequest(http.MethodGet, "/api/s3-instances", nil)
	is.NoErr(err)

	rr := httptest.NewRecorder()
	s3manager.HandleGetS3Instances(instances).ServeHTTP(rr, req)
	resp := rr.Result()
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()

	is.Equal(http.StatusOK, resp.StatusCode)

	var result struct {
		Instances []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"instances"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	is.NoErr(err)

	is.Equal(2, len(result.Instances))
	is.Equal("1", result.Instances[0].ID)
	is.Equal("first", result.Instances[0].Name)
	is.Equal("2", result.Instances[1].ID)
	is.Equal("second", result.Instances[1].Name)
}
