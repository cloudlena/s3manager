package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// HandleBucketsViewWithManager renders all buckets on an HTML page using MultiS3Manager.
func HandleBucketsViewWithManager(manager *MultiS3Manager, templates fs.FS, allowDelete bool, rootURL string) http.HandlerFunc {
	type pageData struct {
		RootURL      string
		Buckets      []interface{}
		AllowDelete  bool
		CurrentS3    *S3Instance
		S3Instances  []*S3Instance
		HasError     bool
		ErrorMessage string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		current := manager.GetCurrentInstance()
		instances := manager.GetAllInstances()

		buckets, err := s3.ListBuckets(r.Context())

		data := pageData{
			RootURL:     rootURL,
			AllowDelete: allowDelete,
			CurrentS3:   current,
			S3Instances: instances,
			HasError:    false,
		}

		if err != nil {
			// Instead of returning an HTTP error, show a user-friendly message
			data.HasError = true
			data.ErrorMessage = fmt.Sprintf("Unable to connect to S3 instance '%s'. Please check the credentials and try switching to another instance.", current.Name)
			data.Buckets = make([]interface{}, 0) // Empty buckets list
		} else {
			data.Buckets = make([]interface{}, len(buckets))
			for i, bucket := range buckets {
				data.Buckets[i] = bucket
			}
		}

		t, err := template.ParseFS(templates, "layout.html.tmpl", "buckets.html.tmpl")
		if err != nil {
			handleHTTPError(w, err)
			return
		}
		err = t.ExecuteTemplate(w, "layout", data)
		if err != nil {
			handleHTTPError(w, err)
			return
		}
	}
}

// HandleBucketViewWithManager shows the details page of a bucket using MultiS3Manager.
func HandleBucketViewWithManager(manager *MultiS3Manager, templates fs.FS, allowDelete bool, listRecursive bool, rootURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		current := manager.GetCurrentInstance()
		instances := manager.GetAllInstances()

		// Create a modified handler that includes S3 instance data
		handler := createBucketViewWithS3Data(s3, templates, allowDelete, listRecursive, rootURL, current, instances)
		handler(w, r)
	}
}

// HandleCreateBucketWithManager creates a new bucket using MultiS3Manager.
func HandleCreateBucketWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		// Delegate to the original handler with the current S3 client
		handler := HandleCreateBucket(s3)
		handler(w, r)
	}
}

// HandleDeleteBucketWithManager deletes a bucket using MultiS3Manager.
func HandleDeleteBucketWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		// Delegate to the original handler with the current S3 client
		handler := HandleDeleteBucket(s3)
		handler(w, r)
	}
}

// HandleCreateObjectWithManager uploads a new object using MultiS3Manager.
func HandleCreateObjectWithManager(manager *MultiS3Manager, sseInfo SSEType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		// Delegate to the original handler with the current S3 client
		handler := HandleCreateObject(s3, sseInfo)
		handler(w, r)
	}
}

// HandleGenerateURLWithManager generates a presigned URL using MultiS3Manager.
func HandleGenerateURLWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		// Delegate to the original handler with the current S3 client
		handler := HandleGenerateURL(s3)
		handler(w, r)
	}
}

// HandleGetObjectWithManager downloads an object to the client using MultiS3Manager.
func HandleGetObjectWithManager(manager *MultiS3Manager, forceDownload bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		// Delegate to the original handler with the current S3 client
		handler := HandleGetObject(s3, forceDownload)
		handler(w, r)
	}
}

// HandleDeleteObjectWithManager deletes an object using MultiS3Manager.
func HandleDeleteObjectWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s3 := manager.GetCurrentClient()
		// Delegate to the original handler with the current S3 client
		handler := HandleDeleteObject(s3)
		handler(w, r)
	}
}

// createBucketViewWithS3Data creates a bucket view handler that includes S3 instance data
func createBucketViewWithS3Data(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool, rootURL string, current *S3Instance, instances []*S3Instance) http.HandlerFunc {
	type objectWithIcon struct {
		Key          string
		Size         int64
		SizeDisplay  string
		LastModified time.Time
		Owner        string
		Icon         string
		IsFolder     bool
		DisplayName  string
	}

	type pageData struct {
		RootURL      string
		BucketName   string
		Objects      []objectWithIcon
		AllowDelete  bool
		Paths        []string
		CurrentPath  string
		CurrentS3    *S3Instance
		S3Instances  []*S3Instance
		HasError     bool
		ErrorMessage string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		regex := regexp.MustCompile(`\/buckets\/([^\/]*)\/?(.*)`)
		matches := regex.FindStringSubmatch(r.RequestURI)
		bucketName := matches[1]
		path := matches[2]

		var objs []objectWithIcon
		hasError := false
		errorMessage := ""

		opts := minio.ListObjectsOptions{
			Recursive: listRecursive,
			Prefix:    path,
		}
		objectCh := s3.ListObjects(r.Context(), bucketName, opts)
		for object := range objectCh {
			if object.Err != nil {
				// Instead of returning HTTP error, show user-friendly message
				hasError = true
				if strings.Contains(object.Err.Error(), "AccessDenied") || strings.Contains(object.Err.Error(), "InvalidAccessKeyId") || strings.Contains(object.Err.Error(), "SignatureDoesNotMatch") {
					errorMessage = fmt.Sprintf("Unable to access bucket '%s' on S3 instance '%s'. Please check the credentials and try switching to another instance.", bucketName, current.Name)
				} else if strings.Contains(object.Err.Error(), ErrBucketDoesNotExist) {
					errorMessage = fmt.Sprintf("Bucket '%s' does not exist on S3 instance '%s'. Please try switching to another instance or go back to the buckets list.", bucketName, current.Name)
				} else {
					errorMessage = fmt.Sprintf("Unable to list objects in bucket '%s' on S3 instance '%s'. Please try switching to another instance.", bucketName, current.Name)
				}
				break
			}

			var sizeDisplay string
			if current != nil && current.HumanReadableSize {
				sizeDisplay = FormatFileSize(object.Size)
			} else {
				sizeDisplay = fmt.Sprintf("%d bytes", object.Size)
			}

			obj := objectWithIcon{
				Key:          object.Key,
				Size:         object.Size,
				SizeDisplay:  sizeDisplay,
				LastModified: object.LastModified,
				Owner:        object.Owner.DisplayName,
				Icon:         icon(object.Key),
				IsFolder:     strings.HasSuffix(object.Key, "/"),
				DisplayName:  strings.TrimSuffix(strings.TrimPrefix(object.Key, path), "/"),
			}
			objs = append(objs, obj)
		}

		data := pageData{
			RootURL:      rootURL,
			BucketName:   bucketName,
			Objects:      objs,
			AllowDelete:  allowDelete,
			Paths:        removeEmptyStrings(strings.Split(path, "/")),
			CurrentPath:  path,
			CurrentS3:    current,
			S3Instances:  instances,
			HasError:     hasError,
			ErrorMessage: errorMessage,
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
