package s3manager

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/pkg/errors"
)

// HandleBucketsView renders all buckets on an HTML page.
func HandleBucketsView(s3 S3, tmplDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buckets, err := s3.ListBuckets()
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error listing buckets"))
			return
		}

		l := filepath.Join(tmplDir, "layout.html.tmpl")
		p := filepath.Join(tmplDir, "buckets.html.tmpl")
		t, err := template.ParseFiles(l, p)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error parsing template files"))
			return
		}
		err = t.ExecuteTemplate(w, "layout", buckets)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error executing template"))
			return
		}
	}
}
