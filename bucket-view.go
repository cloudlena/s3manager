package main

import (
	"html/template"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
)

// ObjectWithIcon is a minio object with an added icon
type ObjectWithIcon struct {
	minio.ObjectInfo
	Icon string
}

// BucketPage defines the details page of a bucket
type BucketPage struct {
	BucketName string
	Objects    []ObjectWithIcon
}

// BucketViewHandler shows the details page of a bucket
func BucketViewHandler(s3 S3Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		var objs []ObjectWithIcon

		l := path.Join("templates", "layout.html.tmpl")
		p := path.Join("templates", "bucket.html.tmpl")

		t, err := template.ParseFiles(l, p)
		if err != nil {
			msg := "error parsing templates"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}

		doneCh := make(chan struct{})
		defer close(doneCh)
		objectCh := s3.ListObjectsV2(bucketName, "", true, doneCh)
		for object := range objectCh {
			if object.Err != nil {
				msg := "error listing objects"
				code := http.StatusInternalServerError
				if object.Err.Error() == "The specified bucket does not exist." {
					msg = "bucket not found"
					code = http.StatusNotFound
				}

				handleHTTPError(w, msg, object.Err, code)
				return
			}
			objectWithIcon := ObjectWithIcon{object, icon(object.Key)}
			objs = append(objs, objectWithIcon)
		}

		bucketPage := BucketPage{
			BucketName: bucketName,
			Objects:    objs,
		}

		err = t.ExecuteTemplate(w, "layout", bucketPage)
		if err != nil {
			msg := "error executing template"
			handleHTTPError(w, msg, err, http.StatusInternalServerError)
			return
		}
	})
}

// icon returns an icon for a file type
func icon(fileName string) string {
	e := path.Ext(fileName)

	switch e {
	case ".tgz", ".gz":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3", ".wav":
		return "music_note"
	}

	return "insert_drive_file"
}
