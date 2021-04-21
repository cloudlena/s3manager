package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"

	"github.com/matryer/way"
	minio "github.com/minio/minio-go"
)

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, templates fs.FS) http.HandlerFunc {
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
				handleHTTPError(w, fmt.Errorf("error listing objects: %w", object.Err))
				return
			}
			obj := objectWithIcon{Info: object, Icon: icon(object.Key)}
			objs = append(objs, obj)
		}
		data := pageData{
			BucketName: bucketName,
			Objects:    objs,
		}

		t, err := template.ParseFS(templates, "layout.html.tmpl", "bucket.html.tmpl")
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
