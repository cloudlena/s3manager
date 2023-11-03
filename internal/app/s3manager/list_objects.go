package s3manager

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"net/http"
	"strings"
	"time"
)

type ListEntry struct {
	IsDir        bool       `json:"isDir"`
	Name         string     `json:"name"`
	Size         int64      `json:"size"`
	LastModified *time.Time `json:"lastModified"`
}

func HandleListObjects(s3 S3) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		prefix := mux.Vars(r)["objectName"]

		if prefix != "" && strings.HasSuffix(prefix, "/") == false {
			prefix += "/"
		}

		objects := s3.ListObjects(r.Context(), bucketName, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: false,
		})

		var fileEntry []ListEntry

		for o := range objects {
			// Iterate over the objects, define directories and files and output them
			if o.Err != nil {
				handleHTTPError(w, o.Err)
				return
			}

			if o.Key == prefix {
				continue
			}

			var lastModified *time.Time
			if o.LastModified.IsZero() {
				lastModified = nil
			} else {
				lastModified = &o.LastModified
			}

			if o.Key[len(o.Key)-1] == '/' {
				fileEntry = append(fileEntry, ListEntry{
					IsDir:        true,
					Name:         o.Key[len(prefix) : len(o.Key)-1],
					LastModified: lastModified,
					Size:         0,
				})
			} else {
				fileEntry = append(fileEntry, ListEntry{
					IsDir:        false,
					Name:         o.Key[len(prefix):],
					LastModified: lastModified,
					Size:         o.Size,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if fileEntry == nil {
			fileEntry = []ListEntry{}
		}
		err := json.NewEncoder(w).Encode(fileEntry)
		if err != nil {
			handleHTTPError(w, err)
			return
		}
	})
}
