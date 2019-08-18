package s3manager

import (
	"html/template"
	"net/http"
	"path"
	"path/filepath"

	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, tmplDir string) http.HandlerFunc {
	type objectWithIcon struct {
		Info minio.ObjectInfo
		Icon string
	}

	type pageData struct {
		BucketName string
		Objects    []objectWithIcon
	}

	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := way.Param(r.Context(), "bucketName")

		var objs []objectWithIcon
		doneCh := make(chan struct{})
		defer close(doneCh)
		objectCh := s3.ListObjectsV2(bucketName, "", true, doneCh)
		for object := range objectCh {
			if object.Err != nil {
				handleHTTPError(w, errors.Wrap(object.Err, "error listing objects"))
				return
			}
			obj := objectWithIcon{Info: object, Icon: icon(object.Key)}
			objs = append(objs, obj)
		}
		data := pageData{
			BucketName: bucketName,
			Objects:    objs,
		}

		l := filepath.Join(tmplDir, "layout.html.tmpl")
		p := filepath.Join(tmplDir, "bucket.html.tmpl")
		t, err := template.ParseFiles(l, p)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error parsing template files"))
			return
		}
		err = t.ExecuteTemplate(w, "layout", data)
		if err != nil {
			handleHTTPError(w, errors.Wrap(err, "error executing template"))
			return
		}
	}
}

// icon returns an icon for a file type.
func icon(fileName string) string {
	e := path.Ext(fileName)
	switch e {
	case ".tgz", ".gz", ".zip":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3", ".wav":
		return "music_note"
	}
	return "insert_drive_file"
}
