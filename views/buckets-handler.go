package views

import (
	"html/template"
	"net/http"
	"path"

	"github.com/mastertinner/s3-manager/web"
	minio "github.com/minio/minio-go"
)

// BucketsHandler shows all buckets
func BucketsHandler(s3 *minio.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lp := path.Join("views", "layout.html")
		p := path.Join("views", "buckets.html")

		t, err := template.ParseFiles(lp, p)
		if err != nil {
			msg := "error parsing templates"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		buckets, err := s3.ListBuckets()
		if err != nil {
			msg := "error listing buckets"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		err = t.ExecuteTemplate(w, "layout", buckets)
		if err != nil {
			msg := "error executing template"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}
