package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/minio/minio-go/v7"
)

// HandleBucketsView renders all buckets on an HTML page.
func HandleBucketsView(s3 S3, templates fs.FS, allowDelete bool, bucketName string) http.HandlerFunc {
	type pageData struct {
		Buckets     []minio.BucketInfo
		AllowDelete bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		buckets, err := s3.ListBuckets(r.Context())
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error listing buckets: %w", err))
			return
		}

		var filteredBuckets []minio.BucketInfo
		if bucketName != "" {
			for _, bucket := range buckets {
				if bucket.Name == bucketName {
					filteredBuckets = append(filteredBuckets, bucket)
					break // Eşleşen ilk bucket bulunduğunda döngüden çık
				}
			}
			buckets = filteredBuckets
		}

		data := pageData{
			Buckets:     buckets,
			AllowDelete: allowDelete,
		}

		t, err := template.ParseFS(templates, "layout.html.tmpl", "buckets.html.tmpl")
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing template files: %w", err))
			return
		}
		err = t.ExecuteTemplate(w, "layout", data)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error executing template: %w", err))
			return
		}
	}
}
