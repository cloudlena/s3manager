package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// objectWithIconExtended extends objectWithIcon with additional formatting fields
type objectWithIconExtended struct {
	Key          string
	Size         int64
	SizeDisplay  string
	LastModified time.Time
	Owner        string
	Icon         string
	IsFolder     bool
	DisplayName  string
}

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
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
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
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		instances := manager.GetAllInstances()

		// Create a modified handler that includes S3 instance data
		handler := createBucketViewWithS3Data(s3, templates, allowDelete, listRecursive, rootURL, current, instances)
		handler(w, r)
	}
}

// HandleCreateBucketWithManager creates a new bucket using MultiS3Manager.
func HandleCreateBucketWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the original handler with the current S3 client
		handler := HandleCreateBucket(s3)
		handler(w, r)
	}
}

// HandleDeleteBucketWithManager deletes a bucket using MultiS3Manager.
func HandleDeleteBucketWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the original handler with the current S3 client
		handler := HandleDeleteBucket(s3)
		handler(w, r)
	}
}

// HandleCreateObjectWithManager uploads a new object using MultiS3Manager.
func HandleCreateObjectWithManager(manager *MultiS3Manager, sseInfo SSEType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the original handler with the current S3 client
		handler := HandleCreateObject(s3, sseInfo)
		handler(w, r)
	}
}

// HandleGenerateURLWithManager generates a presigned URL using MultiS3Manager.
func HandleGenerateURLWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the original handler with the current S3 client
		handler := HandleGenerateURL(s3)
		handler(w, r)
	}
}

// HandleGetObjectWithManager downloads an object to the client using MultiS3Manager.
func HandleGetObjectWithManager(manager *MultiS3Manager, forceDownload bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the original handler with the current S3 client
		handler := HandleGetObject(s3, forceDownload)
		handler(w, r)
	}
}

// HandleDeleteObjectWithManager deletes an object using MultiS3Manager.
func HandleDeleteObjectWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the original handler with the current S3 client
		handler := HandleDeleteObject(s3)
		handler(w, r)
	}
}

// createBucketViewWithS3Data creates a bucket view handler that includes S3 instance data
func createBucketViewWithS3Data(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool, rootURL string, current *S3Instance, instances []*S3Instance) http.HandlerFunc {
	type pageData struct {
		RootURL      string
		BucketName   string
		Objects      []objectWithIconExtended
		AllowDelete  bool
		Paths        []string
		CurrentPath  string
		CurrentS3    *S3Instance
		S3Instances  []*S3Instance
		HasError     bool
		ErrorMessage string
		SortBy       string
		SortOrder    string
		Page         int
		PerPage      int
		TotalItems   int
		TotalPages   int
		HasPrevPage  bool
		HasNextPage  bool
		Search       string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Updated regex to handle instance in the URL path
		regex := regexp.MustCompile(`\/([^\/]+)\/buckets\/([^\/]*)\/?(.*)`)
		matches := regex.FindStringSubmatch(r.URL.Path)
		if len(matches) < 3 {
			handleHTTPError(w, fmt.Errorf("invalid URL path"))
			return
		}
		bucketName := matches[2]
		path := ""
		if len(matches) > 3 {
			path = matches[3]
		}

		// Get sorting parameters from query string
		sortBy := r.URL.Query().Get("sortBy")
		sortOrder := r.URL.Query().Get("sortOrder")

		// Default sorting
		if sortBy == "" {
			sortBy = "key"
		}
		if sortOrder == "" {
			sortOrder = "asc"
		}

		// Get pagination parameters
		page := 1
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		perPage := 25
		showAll := false
		if perPageStr := r.URL.Query().Get("perPage"); perPageStr != "" {
			if pp, err := strconv.Atoi(perPageStr); err == nil {
				if pp == 0 || pp == -1 {
					showAll = true
				} else if pp > 0 {
					perPage = pp
				}
			}
		}

		// Get search parameter
		search := strings.TrimSpace(r.URL.Query().Get("search"))

		var objs []objectWithIconExtended
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

			sizeDisplay := FormatFileSize(object.Size)

			obj := objectWithIconExtended{
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

		// Filter objects based on search query
		if search != "" && !hasError {
			searchLower := strings.ToLower(search)
			filteredObjs := make([]objectWithIconExtended, 0)
			for _, obj := range objs {
				// Search in DisplayName and Key (case-insensitive)
				if strings.Contains(strings.ToLower(obj.DisplayName), searchLower) ||
					strings.Contains(strings.ToLower(obj.Key), searchLower) {
					filteredObjs = append(filteredObjs, obj)
				}
			}
			objs = filteredObjs
		}

		// Sort objects based on sortBy and sortOrder
		if !hasError {
			sortObjectsWithSize(objs, sortBy, sortOrder)
		}

		// Calculate pagination
		totalItems := len(objs)
		totalPages := 1
		if !showAll {
			totalPages = (totalItems + perPage - 1) / perPage
			if totalPages == 0 {
				totalPages = 1
			}
			if page > totalPages {
				page = totalPages
			}
		}

		// Paginate objects
		if showAll {
			// Show all items - no pagination
			perPage = totalItems
			if perPage == 0 {
				perPage = 1 // Avoid division by zero
			}
			page = 1
		} else {
			// Apply pagination
			start := (page - 1) * perPage
			end := start + perPage
			if start < 0 {
				start = 0
			}
			if end > totalItems {
				end = totalItems
			}
			if start < totalItems && !hasError {
				objs = objs[start:end]
			} else if !hasError {
				objs = []objectWithIconExtended{}
			}
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
			SortBy:       sortBy,
			SortOrder:    sortOrder,
			Page:         page,
			PerPage:      perPage,
			TotalItems:   totalItems,
			TotalPages:   totalPages,
			HasPrevPage:  page > 1,
			HasNextPage:  page < totalPages,
			Search:       search,
		}

		funcMap := template.FuncMap{
			"add": func(a, b int) int { return a + b },
			"sub": func(a, b int) int { return a - b },
			"mul": func(a, b int) int { return a * b },
			"min": func(a, b int) int {
				if a < b {
					return a
				}
				return b
			},
			"iterate": func(start, end int) []int {
				result := make([]int, 0, end-start)
				for i := start; i < end; i++ {
					result = append(result, i)
				}
				return result
			},
		}

		t, err := template.New("").Funcs(funcMap).ParseFS(templates, "layout.html.tmpl", "bucket.html.tmpl")
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

// sortObjectsWithSize sorts objects with SizeDisplay field based on the specified field and order
func sortObjectsWithSize(objs []objectWithIconExtended, sortBy, sortOrder string) {
	sort.Slice(objs, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "size":
			less = objs[i].Size < objs[j].Size
		case "owner":
			less = strings.ToLower(objs[i].Owner) < strings.ToLower(objs[j].Owner)
		case "lastModified":
			less = objs[i].LastModified.Before(objs[j].LastModified)
		case "key":
			fallthrough
		default:
			less = strings.ToLower(objs[i].DisplayName) < strings.ToLower(objs[j].DisplayName)
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// HandleBulkDeleteObjectsWithManager deletes multiple objects using MultiS3Manager.
func HandleBulkDeleteObjectsWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the bulk delete handler with the current S3 client
		handler := HandleBulkDeleteObjects(s3)
		handler(w, r)
	}
}

// HandleBulkDownloadObjectsWithManager downloads multiple objects as a ZIP using MultiS3Manager.
func HandleBulkDownloadObjectsWithManager(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceName := vars["instance"]

		current, err := manager.GetInstance(instanceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Instance not found: %s", err.Error()), http.StatusNotFound)
			return
		}

		s3 := current.Client
		// Delegate to the bulk download handler with the current S3 client
		handler := HandleBulkDownloadObjects(s3)
		handler(w, r)
	}
}
