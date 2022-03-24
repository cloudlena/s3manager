package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool) http.HandlerFunc {
	type objectWithIcon struct {
		Info minio.ObjectInfo
		Icon string
	}

	type pageData struct {
		BucketName  string
		Objects     []objectWithIcon
		AllowDelete bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]

		var objs []objectWithIcon
		doneCh := make(chan struct{})
		defer close(doneCh)
		opts := minio.ListObjectsOptions{
			Recursive: listRecursive,
		}
		objectCh := s3.ListObjects(r.Context(), bucketName, opts)
		for object := range objectCh {
			if object.Err != nil {
				handleHTTPError(w, fmt.Errorf("error listing objects: %w", object.Err))
				return
			}
			obj := objectWithIcon{Info: object, Icon: icon(object.Key)}
			objs = append(objs, obj)
		}
		data := pageData{
			BucketName:  bucketName,
			Objects:     objs,
			AllowDelete: allowDelete,
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
