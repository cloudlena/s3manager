package s3manager_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
)

func TestHandleBucketView(t *testing.T) {
	cases := map[string]struct {
		listObjectsV2Func    func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo
		bucketName           string
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"renders a bucket containing a file": {
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
		"renders placeholder for an empty bucket": {
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				close(objCh)
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "No objects in",
		},
		"renders a bucket containing an archive": {
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
		"renders a bucket containing an image": {
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
		"renders a bucket containing a sound file": {
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
		"returns error if the bucket doesn't exist": {
			listObjectsV2Func: func(string, string, bool, <-chan struct{}) <-chan minio.ObjectInfo {
				objCh := make(chan minio.ObjectInfo)
				go func() {
					objCh <- minio.ObjectInfo{Err: s3manager.ErrBucketDoesNotExist}
					close(objCh)
				}()
				return objCh
			},
			bucketName:           "testBucket",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: http.StatusText(http.StatusNotFound),
		},
		"returns error if there is an S3 error": {
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

	for tcID, tc := range cases {
		t.Run(tcID, func(t *testing.T) {
			assert := assert.New(t)

			s3 := &S3Mock{
				ListObjectsV2Func: tc.listObjectsV2Func,
			}

			tmplDir := filepath.Join("..", "..", "..", "web", "template")
			r := way.NewRouter()
			r.Handle(http.MethodGet, "/buckets/:bucketName", s3manager.HandleBucketView(s3, tmplDir))

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := fmt.Sprintf("%s/buckets/%s", ts.URL, tc.bucketName)
			resp, err := http.Get(url)
			assert.NoError(err)
			defer func() {
				err = resp.Body.Close()
				if err != nil {
					t.FailNow()
				}
			}()

			body, err := ioutil.ReadAll(resp.Body)
			assert.NoError(err)

			assert.Equal(tc.expectedStatusCode, resp.StatusCode)
			assert.Contains(string(body), tc.expectedBodyContains)
		})
	}
}
