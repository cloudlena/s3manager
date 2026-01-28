package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
)

// HandleBucketsView renders all buckets on an HTML page.
func HandleBucketsView(s3 S3, templates fs.FS, allowDelete bool, rootURL string) http.HandlerFunc {
	type pageData struct {
		RootURL      string
		Buckets      []interface{}
		AllowDelete  bool
		CurrentS3    *S3Instance
		S3Instances  []*S3Instance
		HasError     bool
		ErrorMessage string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		buckets, err := s3.ListBuckets(r.Context())

		data := pageData{
			RootURL:     rootURL,
			AllowDelete: allowDelete,
			HasError:    false,
		}

		if err != nil {
			handleHTTPError(w, fmt.Errorf("error listing buckets: %w", err))
			return
		}

		data.Buckets = make([]interface{}, len(buckets))
		for i, bucket := range buckets {
			data.Buckets[i] = bucket
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
