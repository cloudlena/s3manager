package s3manager_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/cloudlena/s3manager/internal/app/s3manager/mocks"
	"github.com/gorilla/mux"
	"github.com/matryer/is"
	"github.com/minio/minio-go/v7"
)

// newSingleInstanceManager wraps a mock client in a MultiS3Manager with one
// instance whose ID is "1".
func newSingleInstanceManager(client s3manager.S3) *s3manager.MultiS3Manager {
	return s3manager.NewMultiS3ManagerWithInstances([]*s3manager.S3Instance{
		{ID: "1", Name: "test", Client: client},
	})
}

// doMove issues a move/copy request to the handler with the given source vars
// and JSON body.
func doMove(t *testing.T, manager *s3manager.MultiS3Manager, bucket, object, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/api/buckets/"+bucket+"/objects/"+object+"/move", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"bucketName": bucket, "objectName": object})
	rr := httptest.NewRecorder()
	s3manager.HandleMoveObjectWithManager(manager, s3manager.SSEType{}).ServeHTTP(rr, req)
	return rr
}

func TestHandleMoveObjectSameInstance(t *testing.T) {
	t.Parallel()

	t.Run("copies a file without deleting the source", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		s3 := &mocks.S3Mock{
			CopyObjectFunc: func(context.Context, minio.CopyDestOptions, minio.CopySrcOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, nil
			},
		}
		manager := newSingleInstanceManager(s3)

		rr := doMove(t, manager, "src", "photos/cat.jpg",
			`{"destBucket":"dst","destPath":"archive/","operation":"copy"}`)

		is.Equal(http.StatusOK, rr.Code)
		is.Equal(len(s3.CopyObjectCalls()), 1)
		is.Equal(len(s3.RemoveObjectCalls()), 0) // copy must not delete the source
		call := s3.CopyObjectCalls()[0]
		is.Equal(call.Dst.Bucket, "dst")
		is.Equal(call.Dst.Object, "archive/cat.jpg") // keeps original name under prefix
		is.Equal(call.Src.Bucket, "src")
		is.Equal(call.Src.Object, "photos/cat.jpg")
	})

	t.Run("moves a file and deletes the source", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		s3 := &mocks.S3Mock{
			CopyObjectFunc: func(context.Context, minio.CopyDestOptions, minio.CopySrcOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, nil
			},
			RemoveObjectFunc: func(context.Context, string, string, minio.RemoveObjectOptions) error {
				return nil
			},
		}
		manager := newSingleInstanceManager(s3)

		rr := doMove(t, manager, "src", "photos/cat.jpg",
			`{"destBucket":"dst","destPath":"archive","operation":"move"}`)

		is.Equal(http.StatusOK, rr.Code)
		is.Equal(len(s3.CopyObjectCalls()), 1)
		is.Equal(len(s3.RemoveObjectCalls()), 1)
		is.Equal(s3.RemoveObjectCalls()[0].ObjectName, "photos/cat.jpg")
	})

	t.Run("moves a folder recursively", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		s3 := &mocks.S3Mock{
			ListObjectsFunc: func(context.Context, string, minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				ch := make(chan minio.ObjectInfo, 2)
				ch <- minio.ObjectInfo{Key: "photos/a.jpg"}
				ch <- minio.ObjectInfo{Key: "photos/sub/b.jpg"}
				close(ch)
				return ch
			},
			CopyObjectFunc: func(context.Context, minio.CopyDestOptions, minio.CopySrcOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, nil
			},
			RemoveObjectFunc: func(context.Context, string, string, minio.RemoveObjectOptions) error {
				return nil
			},
		}
		manager := newSingleInstanceManager(s3)

		rr := doMove(t, manager, "src", "photos/",
			`{"destBucket":"dst","destPath":"backup/","operation":"move"}`)

		is.Equal(http.StatusOK, rr.Code)
		is.Equal(len(s3.CopyObjectCalls()), 2)
		is.Equal(len(s3.RemoveObjectCalls()), 2)
		// Folder name is preserved under the destination prefix.
		is.Equal(s3.CopyObjectCalls()[0].Dst.Object, "backup/photos/a.jpg")
		is.Equal(s3.CopyObjectCalls()[1].Dst.Object, "backup/photos/sub/b.jpg")
	})

	t.Run("rejects a request without a destination bucket", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		s3 := &mocks.S3Mock{}
		manager := newSingleInstanceManager(s3)

		rr := doMove(t, manager, "src", "photos/cat.jpg",
			`{"destPath":"archive/","operation":"move"}`)

		is.Equal(http.StatusBadRequest, rr.Code)
	})

	t.Run("rejects an in-place move", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		s3 := &mocks.S3Mock{}
		manager := newSingleInstanceManager(s3)

		rr := doMove(t, manager, "src", "photos/cat.jpg",
			`{"destBucket":"src","destPath":"photos/","operation":"move"}`)

		is.Equal(http.StatusBadRequest, rr.Code)
		is.Equal(len(s3.CopyObjectCalls()), 0)
	})

	t.Run("returns an error when the copy fails", func(t *testing.T) {
		t.Parallel()
		is := is.New(t)

		s3 := &mocks.S3Mock{
			CopyObjectFunc: func(context.Context, minio.CopyDestOptions, minio.CopySrcOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, errS3
			},
		}
		manager := newSingleInstanceManager(s3)

		rr := doMove(t, manager, "src", "photos/cat.jpg",
			`{"destBucket":"dst","destPath":"archive/","operation":"move"}`)

		is.Equal(http.StatusInternalServerError, rr.Code)
		is.Equal(len(s3.RemoveObjectCalls()), 0) // source must survive a failed copy
	})
}
