package s3manager

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

// newTestMultiS3Manager constructs a MultiS3Manager directly from S3Instance slices,
// bypassing NewMultiS3Manager so tests can inject mock S3 clients.
func newTestMultiS3Manager(instances []*S3Instance) *MultiS3Manager {
	m := &MultiS3Manager{
		instances:     make(map[string]*S3Instance),
		instanceOrder: make([]string, 0, len(instances)),
	}
	for _, inst := range instances {
		m.instances[inst.ID] = inst
		m.instanceOrder = append(m.instanceOrder, inst.ID)
	}
	return m
}

// stubS3 is a minimal S3 implementation for use in manager handler tests.
// Only implement the methods needed; the rest panic so we notice if they're called unexpectedly.
type stubS3 struct {
	listBuckets        func(context.Context) ([]minio.BucketInfo, error)
	makeBucket         func(context.Context, string, minio.MakeBucketOptions) error
	removeBucket       func(context.Context, string) error
	removeObject       func(context.Context, string, string, minio.RemoveObjectOptions) error
	getBucketPolicy    func(context.Context, string) (string, error)
	setBucketPolicy    func(context.Context, string, string) error
	removeObjects      func(context.Context, string, <-chan minio.ObjectInfo, minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
	listObjects        func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo
	endpointURL        func() *url.URL
	presignedGetObject func(context.Context, string, string, time.Duration, url.Values) (*url.URL, error)
}

func (s *stubS3) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return s.listBuckets(ctx)
}
func (s *stubS3) MakeBucket(ctx context.Context, name string, opts minio.MakeBucketOptions) error {
	return s.makeBucket(ctx, name, opts)
}
func (s *stubS3) RemoveBucket(ctx context.Context, name string) error {
	return s.removeBucket(ctx, name)
}
func (s *stubS3) RemoveObject(ctx context.Context, bucket, object string, opts minio.RemoveObjectOptions) error {
	return s.removeObject(ctx, bucket, object, opts)
}
func (s *stubS3) GetBucketPolicy(ctx context.Context, bucket string) (string, error) {
	return s.getBucketPolicy(ctx, bucket)
}
func (s *stubS3) SetBucketPolicy(ctx context.Context, bucket, policy string) error {
	return s.setBucketPolicy(ctx, bucket, policy)
}
func (s *stubS3) RemoveObjects(ctx context.Context, bucket string, ch <-chan minio.ObjectInfo, opts minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
	return s.removeObjects(ctx, bucket, ch, opts)
}
func (s *stubS3) ListObjects(ctx context.Context, bucket string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	return s.listObjects(ctx, bucket, opts)
}
func (s *stubS3) EndpointURL() *url.URL {
	return s.endpointURL()
}
func (s *stubS3) GetObject(_ context.Context, _, _ string, _ minio.GetObjectOptions) (*minio.Object, error) {
	panic("GetObject not expected in this test")
}
func (s *stubS3) PresignedGetObject(ctx context.Context, bucket, object string, expiry time.Duration, params url.Values) (*url.URL, error) {
	if s.presignedGetObject != nil {
		return s.presignedGetObject(ctx, bucket, object, expiry, params)
	}
	panic("PresignedGetObject not expected in this test")
}
func (s *stubS3) PutObject(_ context.Context, _, _ string, _ io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
	panic("PutObject not expected in this test")
}
func (s *stubS3) StatObject(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	panic("StatObject not expected in this test")
}

var errManagerTest = errors.New("manager test error")

func TestHandleBucketsViewWithManager(t *testing.T) {
	t.Parallel()

	templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))

	cases := []struct {
		it                   string
		instanceName         string
		listBucketsFunc      func(context.Context) ([]minio.BucketInfo, error)
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return nil, nil
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "renders buckets for valid instance",
			instanceName: "1",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return []minio.BucketInfo{{Name: "MY-BUCKET"}}, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "MY-BUCKET",
		},
		{
			it:           "shows error message when S3 is unreachable",
			instanceName: "1",
			listBucketsFunc: func(context.Context) ([]minio.BucketInfo, error) {
				return nil, errManagerTest
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "Unable to connect",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{listBuckets: tc.listBucketsFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/buckets", HandleBucketsViewWithManager(manager, templates, true, "", "")).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/buckets")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleCreateBucketWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		makeBucketFunc       func(context.Context, string, minio.MakeBucketOptions) error
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			makeBucketFunc: func(context.Context, string, minio.MakeBucketOptions) error {
				return nil
			},
			body:                 `{"name":"test-bucket"}`,
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "creates bucket via valid instance",
			instanceName: "1",
			makeBucketFunc: func(context.Context, string, minio.MakeBucketOptions) error {
				return nil
			},
			body:               `{"name":"test-bucket"}`,
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{makeBucket: tc.makeBucketFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets", HandleCreateBucketWithManager(manager)).Methods(http.MethodPost)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/"+tc.instanceName+"/api/buckets", bytes.NewBufferString(tc.body))
			is.NoErr(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleDeleteBucketWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		removeBucketFunc     func(context.Context, string) error
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			removeBucketFunc: func(context.Context, string) error {
				return nil
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "deletes bucket via valid instance",
			instanceName: "1",
			removeBucketFunc: func(context.Context, string) error {
				return nil
			},
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{removeBucket: tc.removeBucketFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}", HandleDeleteBucketWithManager(manager)).Methods(http.MethodDelete)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodDelete, ts.URL+"/"+tc.instanceName+"/api/buckets/test-bucket", nil)
			is.NoErr(err)

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleDeleteObjectWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		removeObjectFunc     func(context.Context, string, string, minio.RemoveObjectOptions) error
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			removeObjectFunc: func(context.Context, string, string, minio.RemoveObjectOptions) error {
				return nil
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "deletes object via valid instance",
			instanceName: "1",
			removeObjectFunc: func(context.Context, string, string, minio.RemoveObjectOptions) error {
				return nil
			},
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{removeObject: tc.removeObjectFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName}", HandleDeleteObjectWithManager(manager)).Methods(http.MethodDelete)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodDelete, ts.URL+"/"+tc.instanceName+"/api/buckets/test-bucket/objects/test-object", nil)
			is.NoErr(err)

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleGetBucketPolicyWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		getBucketPolicyFunc  func(context.Context, string) (string, error)
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			getBucketPolicyFunc: func(context.Context, string) (string, error) {
				return "", nil
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "gets bucket policy via valid instance",
			instanceName: "1",
			getBucketPolicyFunc: func(context.Context, string) (string, error) {
				return `{"Version":"2012-10-17"}`, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: `{"Version":"2012-10-17"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{getBucketPolicy: tc.getBucketPolicyFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/policy", HandleGetBucketPolicyWithManager(manager)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/api/buckets/test-bucket/policy")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandlePutBucketPolicyWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		setBucketPolicyFunc  func(context.Context, string, string) error
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			setBucketPolicyFunc: func(context.Context, string, string) error {
				return nil
			},
			body:                 `{"Version":"2012-10-17"}`,
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "sets bucket policy via valid instance",
			instanceName: "1",
			setBucketPolicyFunc: func(context.Context, string, string) error {
				return nil
			},
			body:               `{"Version":"2012-10-17"}`,
			expectedStatusCode: http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{setBucketPolicy: tc.setBucketPolicyFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/policy", HandlePutBucketPolicyWithManager(manager)).Methods(http.MethodPut)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPut, ts.URL+"/"+tc.instanceName+"/api/buckets/test-bucket/policy", bytes.NewBufferString(tc.body))
			is.NoErr(err)

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleBulkDeleteObjectsWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		removeObjectsFunc    func(context.Context, string, <-chan minio.ObjectInfo, minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			removeObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
				errCh := make(chan minio.RemoveObjectError)
				go func() {
					defer close(errCh)
					for range objectsCh {
					}
				}()
				return errCh
			},
			body:                 `{"keys":["file1.txt"]}`,
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "bulk deletes via valid instance",
			instanceName: "1",
			removeObjectsFunc: func(_ context.Context, _ string, objectsCh <-chan minio.ObjectInfo, _ minio.RemoveObjectsOptions) <-chan minio.RemoveObjectError {
				errCh := make(chan minio.RemoveObjectError)
				go func() {
					defer close(errCh)
					for range objectsCh {
					}
				}()
				return errCh
			},
			body:                 `{"keys":["file1.txt"]}`,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: `"success": true`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{removeObjects: tc.removeObjectsFunc}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/objects/bulk-delete", HandleBulkDeleteObjectsWithManager(manager)).Methods(http.MethodPost)

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodPost, ts.URL+"/"+tc.instanceName+"/api/buckets/test-bucket/objects/bulk-delete", bytes.NewBufferString(tc.body))
			is.NoErr(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleBulkDownloadObjectsWithManager_NotFound(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	manager := newTestMultiS3Manager([]*S3Instance{
		{ID: "1", Name: "primary", Client: &stubS3{}},
	})

	r := mux.NewRouter()
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/bulk-download", HandleBulkDownloadObjectsWithManager(manager)).Methods(http.MethodGet)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/unknown/api/buckets/test-bucket/objects/bulk-download?keys=%5B%5D")
	is.NoErr(err)
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()
	body, err := io.ReadAll(resp.Body)
	is.NoErr(err)

	is.Equal(http.StatusNotFound, resp.StatusCode)
	is.True(strings.Contains(string(body), "Instance not found"))
}

func TestHandleGenerateURLWithManager(t *testing.T) {
	t.Parallel()

	presignedURL, _ := url.Parse("https://s3.example.com/bucket/object?sig=abc")

	cases := []struct {
		it                     string
		instanceName           string
		presignedGetObjectFunc func(context.Context, string, string, time.Duration, url.Values) (*url.URL, error)
		expectedStatusCode     int
		expectedBodyContains   string
	}{
		{
			it:           "returns 404 for unknown instance",
			instanceName: "unknown",
			presignedGetObjectFunc: func(context.Context, string, string, time.Duration, url.Values) (*url.URL, error) {
				return nil, nil
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
		{
			it:           "generates URL via valid instance",
			instanceName: "1",
			presignedGetObjectFunc: func(context.Context, string, string, time.Duration, url.Values) (*url.URL, error) {
				return presignedURL, nil
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "s3.example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{}
			s3mock.presignedGetObject = tc.presignedGetObjectFunc
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName}/url", HandleGenerateURLWithManager(manager)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/api/buckets/test-bucket/objects/test-object/url?expiry=3600")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleCheckPublicAccessWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:                   "returns 404 for unknown instance",
			instanceName:         "unknown",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{
				endpointURL: func() *url.URL {
					u, _ := url.Parse("http://localhost:9000")
					return u
				},
			}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/public-access", HandleCheckPublicAccessWithManager(manager))

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/api/buckets/test-bucket/objects/test-object/public-access")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleGetObjectMetadataWithManager(t *testing.T) {
	t.Parallel()

	cases := []struct {
		it                   string
		instanceName         string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it:                   "returns 404 for unknown instance",
			instanceName:         "unknown",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "Instance not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			s3mock := &stubS3{
				endpointURL: func() *url.URL {
					u, _ := url.Parse("http://localhost:9000")
					return u
				},
			}
			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: s3mock},
			})

			r := mux.NewRouter()
			r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/metadata", HandleGetObjectMetadataWithManager(manager))

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/" + tc.instanceName + "/api/buckets/test-bucket/objects/test-object/metadata")
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			is.True(strings.Contains(string(body), tc.expectedBodyContains))
		})
	}
}

func TestHandleGetObjectWithManager(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	manager := newTestMultiS3Manager([]*S3Instance{
		{ID: "1", Name: "primary", Client: &stubS3{}},
	})

	r := mux.NewRouter()
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName}", HandleGetObjectWithManager(manager, true)).Methods(http.MethodGet)

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/unknown/api/buckets/test-bucket/objects/test-object")
	is.NoErr(err)
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()
	body, err := io.ReadAll(resp.Body)
	is.NoErr(err)

	is.Equal(http.StatusNotFound, resp.StatusCode)
	is.True(strings.Contains(string(body), "Instance not found"))
}

func TestHandleCreateObjectWithManager(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	manager := newTestMultiS3Manager([]*S3Instance{
		{ID: "1", Name: "primary", Client: &stubS3{}},
	})

	r := mux.NewRouter()
	r.Handle("/{instance}/api/buckets/{bucketName}/objects", HandleCreateObjectWithManager(manager, SSEType{})).Methods(http.MethodPost)

	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/unknown/api/buckets/test-bucket/objects", bytes.NewBufferString("body"))
	is.NoErr(err)

	resp, err := http.DefaultClient.Do(req)
	is.NoErr(err)
	defer func() {
		err = resp.Body.Close()
		is.NoErr(err)
	}()
	body, err := io.ReadAll(resp.Body)
	is.NoErr(err)

	is.Equal(http.StatusNotFound, resp.StatusCode)
	is.True(strings.Contains(string(body), "Instance not found"))
}

func TestHandleBucketViewWithManager(t *testing.T) {
	t.Parallel()

	versionedListObjects := func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
		ch := make(chan minio.ObjectInfo)
		go func() {
			defer close(ch)
			if !opts.WithVersions {
				return
			}
			ch <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v2-abcdefghijk", IsLatest: true}
			ch <- minio.ObjectInfo{Key: "FILE-NAME", VersionID: "v1-abcdefghijk", IsLatest: false}
		}()
		return ch
	}

	cases := []struct {
		it                   string
		path                 string
		client               S3
		showVersions         bool
		expectedStatusCode   int
		expectedBodyContains []string
		unexpectedInBody     []string
	}{
		{
			it:                 "returns not found for an unknown instance",
			path:               "/unknown/buckets/test-bucket/",
			client:             &stubS3{},
			expectedStatusCode: http.StatusNotFound,
			expectedBodyContains: []string{
				"Instance not found",
			},
		},
		{
			it:   "does not show version columns when ShowVersions is disabled",
			path: "/primary/buckets/test-bucket/",
			client: &stubS3{
				listObjects: versionedListObjects,
				endpointURL: func() *url.URL { u, _ := url.Parse("http://localhost:9000"); return u },
			},
			showVersions:       false,
			expectedStatusCode: http.StatusOK,
			unexpectedInBody: []string{
				"Version ID",
				"v1-abcdef",
				"v2-abcdef",
			},
		},
		{
			it:   "renders multiple versions when ShowVersions is enabled",
			path: "/primary/buckets/test-bucket/",
			client: &stubS3{
				listObjects: versionedListObjects,
				endpointURL: func() *url.URL { u, _ := url.Parse("http://localhost:9000"); return u },
			},
			showVersions:       true,
			expectedStatusCode: http.StatusOK,
			expectedBodyContains: []string{
				"Version ID",
				"v1-abcdef",
				"v2-abcdef",
				"Latest",
			},
		},
		{
			it:   "falls back to a normal listing and shows a notice when the provider rejects listing versions",
			path: "/primary/buckets/test-bucket/",
			client: &stubS3{
				listObjects: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
					ch := make(chan minio.ObjectInfo)
					go func() {
						defer close(ch)
						if opts.WithVersions {
							ch <- minio.ObjectInfo{Err: errManagerTest}
							return
						}
						ch <- minio.ObjectInfo{Key: "FILE-NAME"}
					}()
					return ch
				},
				endpointURL: func() *url.URL { u, _ := url.Parse("http://localhost:9000"); return u },
			},
			showVersions:       true,
			expectedStatusCode: http.StatusOK,
			expectedBodyContains: []string{
				"FILE-NAME",
				"Object versions unavailable",
			},
			unexpectedInBody: []string{
				"Version ID",
			},
		},
		{
			it:   "surfaces the underlying S3 error when listing fails for a reason other than versioning",
			path: "/primary/buckets/test-bucket/",
			client: &stubS3{
				listObjects: func(_ context.Context, _ string, _ minio.ListObjectsOptions) <-chan minio.ObjectInfo {
					ch := make(chan minio.ObjectInfo)
					go func() {
						defer close(ch)
						ch <- minio.ObjectInfo{Err: errManagerTest}
					}()
					return ch
				},
				endpointURL: func() *url.URL { u, _ := url.Parse("http://localhost:9000"); return u },
			},
			showVersions:       false,
			expectedStatusCode: http.StatusOK,
			expectedBodyContains: []string{
				errManagerTest.Error(),
			},
		},
		{
			it:   "falls back to a normal listing when the versioned listing succeeds but returns nothing",
			path: "/primary/buckets/test-bucket/",
			client: &stubS3{
				listObjects: func(_ context.Context, _ string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
					ch := make(chan minio.ObjectInfo)
					go func() {
						defer close(ch)
						if opts.WithVersions {
							// Some S3-compatible providers silently return an
							// empty result instead of erroring when versioned
							// listing isn't supported.
							return
						}
						ch <- minio.ObjectInfo{Key: "FILE-NAME"}
					}()
					return ch
				},
				endpointURL: func() *url.URL { u, _ := url.Parse("http://localhost:9000"); return u },
			},
			showVersions:       true,
			expectedStatusCode: http.StatusOK,
			expectedBodyContains: []string{
				"FILE-NAME",
			},
			unexpectedInBody: []string{
				"Version ID",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			t.Parallel()
			is := is.New(t)

			templates := os.DirFS(filepath.Join("..", "..", "..", "web", "template"))

			manager := newTestMultiS3Manager([]*S3Instance{
				{ID: "1", Name: "primary", Client: tc.client},
			})

			r := mux.NewRouter()
			r.PathPrefix("/{instance}/buckets/").Handler(HandleBucketViewWithManager(manager, templates, true, true, "", tc.showVersions)).Methods(http.MethodGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(ts.URL + tc.path)
			is.NoErr(err)
			defer func() {
				err = resp.Body.Close()
				is.NoErr(err)
			}()
			body, err := io.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)
			for _, expected := range tc.expectedBodyContains {
				is.True(strings.Contains(string(body), expected))
			}
			for _, unexpected := range tc.unexpectedInBody {
				is.True(!strings.Contains(string(body), unexpected))
			}
		})
	}
}
