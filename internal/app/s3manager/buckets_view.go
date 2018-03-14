package s3manager

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/pkg/errors"
)

// BucketsViewHandler renders all buckets on an HTML page.
func BucketsViewHandler(s3 S3, tmplDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buckets, err := s3.ListBuckets()
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error listing buckets"))
			return
		}

		l := filepath.Join(tmplDir, "layout.html.tmpl")
		p := filepath.Join(tmplDir, "buckets.html.tmpl")
		t, err := template.ParseFiles(l, p)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errParsingTemplates))
			return
		}
		err = t.ExecuteTemplate(w, "layout", buckets)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errExecutingTemplate))
			return
		}
	})
}
