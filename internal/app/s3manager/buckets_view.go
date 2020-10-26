package s3manager

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

// HandleBucketsView renders all buckets on an HTML page.
func HandleBucketsView(s3 S3, tmplDir string, basepath string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		buckets, err := s3.ListBuckets()
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error listing buckets: %w", err))
			return
		}

		l := filepath.Join(tmplDir, "layout.html.tmpl")
		p := filepath.Join(tmplDir, "buckets.html.tmpl")
		t, err := template.New("").Funcs(template.FuncMap{
			"basepath": func() string {
			  return basepath
			},
		  }).ParseFiles(l, p)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error parsing template files: %w", err))
			return
		}
		err = t.ExecuteTemplate(w, "layout", buckets)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error executing template: %w", err))
			return
		}
	}
}
