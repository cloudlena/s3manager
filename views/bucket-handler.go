package views

import (
	"html/template"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/mastertinner/s3-manager/objects"
	"github.com/mastertinner/s3-manager/web"
	minio "github.com/minio/minio-go"
)

// BucketPage defines the details page of a bucket
type BucketPage struct {
	BucketName string
	Objects    []objects.WithIcon
}

// BucketHandler shows the details page of a bucket
func BucketHandler(s3 *minio.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		var objs []objects.WithIcon

		l := path.Join("views", "layout.html")
		p := path.Join("views", "bucket.html")

		t, err := template.ParseFiles(l, p)
		if err != nil {
			msg := "error parsing templates"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		doneCh := make(chan struct{})
		objectCh := s3.ListObjectsV2(bucketName, "", true, doneCh)
		for object := range objectCh {
			if object.Err != nil {
				msg := "error listing objects"
				web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
				return
			}
			objectWithIcon := objects.WithIcon{object, icon(object.Key)}
			objs = append(objs, objectWithIcon)
		}

		bucketPage := BucketPage{
			BucketName: bucketName,
			Objects:    objs,
		}

		err = t.ExecuteTemplate(w, "layout", bucketPage)
		if err != nil {
			msg := "error executing template"
			web.HandleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}
