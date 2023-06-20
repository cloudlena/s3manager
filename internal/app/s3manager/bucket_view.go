package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool) http.HandlerFunc {
	type objectWithIcon struct {
		Key          string
		Size         int64
		LastModified time.Time
		Owner        string
		Icon         string
		IsFolder     bool
		DisplayName  string
	}

	type pageData struct {
		BucketName  string
		Objects     []objectWithIcon
		AllowDelete bool
		Paths       []string
		CurrentPath string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		regex := regexp.MustCompile(`\/buckets\/([^\/]*)\/?(.*)`)
		matches := regex.FindStringSubmatch(r.RequestURI)
		bucketName := matches[1]
		path := matches[2]

		var objs []objectWithIcon
		doneCh := make(chan struct{})
		defer close(doneCh)
		opts := minio.ListObjectsOptions{
			Recursive: listRecursive,
			Prefix:    path,
		}
		objectCh := s3.ListObjects(r.Context(), bucketName, opts)
		for object := range objectCh {
			if object.Err != nil {
				handleHTTPError(w, fmt.Errorf("error listing objects: %w", object.Err))
				return
			}

			obj := objectWithIcon{
				Key:          object.Key,
				Size:         object.Size,
				LastModified: object.LastModified,
				Owner:        object.Owner.DisplayName,
				Icon:         icon(object.Key),
				IsFolder:     strings.HasSuffix(object.Key, "/"),
				DisplayName:  strings.TrimSuffix(strings.TrimPrefix(object.Key, path), "/"),
			}
			objs = append(objs, obj)
		}
		data := pageData{
			BucketName:  bucketName,
			Objects:     objs,
			AllowDelete: allowDelete,
			Paths:       removeEmptyStrings(strings.Split(path, "/")),
			CurrentPath: path,
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
	if strings.HasSuffix(fileName, "/") {
		return "folder"
	}

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

func removeEmptyStrings(input []string) []string {
	var result []string
	for _, str := range input {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}
