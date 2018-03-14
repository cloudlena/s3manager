package s3manager

import (
	"html/template"
	"net/http"
	"path"
	"path/filepath"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// objectWithIcon is a minio object with an added icon.
type objectWithIcon struct {
	minio.ObjectInfo
	Icon string
}

// bucketPage defines the details page of a bucket.
type bucketPage struct {
	BucketName string
	Objects    []objectWithIcon
}

// BucketViewHandler shows the details page of a bucket.
func BucketViewHandler(s3 S3, tmplDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		var objs []objectWithIcon
		doneCh := make(chan struct{})
		defer close(doneCh)
		objectCh := s3.ListObjectsV2(bucketName, "", true, doneCh)
		for object := range objectCh {
			if object.Err != nil {
				handleHTTPError(w, errors.Wrap(object.Err, "error listing objects"))
				return
			}
			obj := objectWithIcon{object, icon(object.Key)}
			objs = append(objs, obj)
		}
		page := bucketPage{
			BucketName: bucketName,
			Objects:    objs,
		}

		l := filepath.Join(tmplDir, "layout.html.tmpl")
		p := filepath.Join(tmplDir, "bucket.html.tmpl")
		t, err := template.ParseFiles(l, p)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errParsingTemplates))
			return
		}
		err = t.ExecuteTemplate(w, "layout", page)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, errExecutingTemplate))
			return
		}
	})
}

// icon returns an icon for a file type.
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
