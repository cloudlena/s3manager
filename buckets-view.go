package main

import (
	"html/template"
	"net/http"
	"path"
)

// BucketsViewHandler shows all buckets
func BucketsViewHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := path.Join(tmplDirectory, "layout.html.tmpl")
		p := path.Join(tmplDirectory, "buckets.html.tmpl")

		t, err := template.ParseFiles(l, p)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		buckets, err := s3.ListBuckets()
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		err = t.ExecuteTemplate(w, "layout", buckets)
		if err != nil {
			handleHTTPError(w, http.StatusInternalServerError, err)
			return
		}
	})
}
