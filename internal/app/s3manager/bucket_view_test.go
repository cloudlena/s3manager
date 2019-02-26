package s3manager_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/mastertinner/s3manager/internal/app/s3manager/mocks"
	"github.com/matryer/is"
	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
)

func TestHandleBucketView(t *testing.T) {
	cases := []struct {
		it                   string
		listObjectsV2Func    func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo
		bucketName           string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		{
			it: "renders a bucket containing a file",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "testFile"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "testFile",
		},
		{
			it: "renders placeholder for an empty bucket",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				close(objCh)
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No objects in",
		},
		{
			it: "renders a bucket containing an archive",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "archive.tar.gz"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "archive",
		},
		{
			it: "renders a bucket containing an image",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "testImage.png"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "photo",
		},
		{
			it: "renders a bucket containing a sound file",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Key: "testSound.mp3"}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "music_note",
		},
		{
			it: "returns error if the bucket doesn't exist",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Err: errors.New("The specified bucket does not exist")}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: http.StatusText(http.StatusNotFound),
		},
		{
			it: "returns error if there is an S3 error",
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Err: errors.New("mocked S3 error")}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: http.StatusText(http.StatusInternalServerError),
		},
	}

	for _, tc := range cases {
		t.Run(tc.it, func(t *testing.T) {
			is := is.New(t)

			s3 := &mocks.S3Mock{
				ListObjectsV2Func: tc.listObjectsV2Func,
			}

			tmplDir := filepath.Join("..", "..", "..", "web", "template")
			r := way.NewRouter()
			r.Handle(http.MethodGet, "/buckets/:bucketName", s3manager.HandleBucketView(s3, tmplDir))

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, err := http.Get(fmt.Sprintf("%s/buckets/%s", ts.URL, tc.bucketName))
			is.NoErr(err)
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			is.NoErr(err)

			is.Equal(tc.expectedStatusCode, resp.StatusCode)                 // status code
			is.True(strings.Contains(string(body), tc.expectedBodyContains)) // body
		})
	}
}
