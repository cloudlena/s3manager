package s3manager

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

type pathDirectory struct {
	DisplayName string
	Key         string
}

type objectWithIcon struct {
	Key          string
	Size         int64
	LastModified time.Time
	Owner        string
	Icon         string
	IsFolder     bool
	DisplayName  string
}

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool) http.HandlerFunc {

	type pageData struct {
		BucketName      string
		Objects         []objectWithIcon
		AllowDelete     bool
		PathDirectories []pathDirectory
		CurrentPath     string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		pathKey := mux.Vars(r)["pathKey"]

		path, decodeError := decodeVariable(pathKey)
		if decodeError != nil {
			handleHTTPError(w, fmt.Errorf("error when decoding object name: %w", decodeError))
			return
		}

		objectCh := s3.ListObjects(r.Context(), bucketName, minio.ListObjectsOptions{
			Recursive: listRecursive,
			Prefix:    path,
		})
		objs, loadingObjsErr := parseBucketObjects(objectCh, path)
		if loadingObjsErr != nil {
			handleHTTPError(w, fmt.Errorf("error listing objects: %w", loadingObjsErr))
			return
		}

		data := pageData{
			BucketName:      bucketName,
			Objects:         objs,
			AllowDelete:     allowDelete,
			PathDirectories: createPathDirecties(path),
			CurrentPath:     path,
		}

		template, parseTemplateErr := template.ParseFS(templates, "layout.html.tmpl", "bucket.html.tmpl")
		if parseTemplateErr != nil {
			handleHTTPError(w, fmt.Errorf("error parsing template files: %w", parseTemplateErr))
			return
		}

		executeTemplateErr := template.ExecuteTemplate(w, "layout", data)
		if executeTemplateErr != nil {
			handleHTTPError(w, fmt.Errorf("error executing template: %w", executeTemplateErr))
			return
		}
	}
}

func parseBucketObjects(objectCh <-chan minio.ObjectInfo, path string) ([]objectWithIcon, error) {
	var objs []objectWithIcon

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		obj := objectWithIcon{
			Key:          base64.StdEncoding.EncodeToString([]byte(object.Key)),
			Size:         object.Size,
			LastModified: object.LastModified,
			Owner:        object.Owner.DisplayName,
			Icon:         icon(object.Key),
			IsFolder:     strings.HasSuffix(object.Key, "/"),
			DisplayName:  strings.TrimSuffix(strings.TrimPrefix(object.Key, path), "/"),
		}
		objs = append(objs, obj)
	}

	return objs, nil
}

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

func createPathDirecties(path string) []pathDirectory {
	splitPaths := removeEmptyStrings(strings.Split(path, "/"))
	var curDirectoryPath bytes.Buffer
	var result []pathDirectory

	for _, splitPath := range splitPaths {
		curDirectoryPath.WriteString(splitPath)
		curDirectoryPath.WriteByte('/')

		directoryPathKey := base64.StdEncoding.EncodeToString(curDirectoryPath.Bytes())
		result = append(result, pathDirectory{DisplayName: splitPath, Key: directoryPathKey})
	}

	return result
}

func removeEmptyStrings(input []string) []string {
	if len(input) == 0 {
		return input
	}

	var result []string
	for _, str := range input {
		if str != "" {
			result = append(result, str)
		}
	}

	return result
}
