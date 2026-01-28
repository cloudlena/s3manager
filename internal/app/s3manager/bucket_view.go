package s3manager

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// objectWithIcon represents an S3 object with additional display properties
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
func HandleBucketView(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool, rootURL string) http.HandlerFunc {
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
		regex := regexp.MustCompile(`\/buckets\/([^\/]*)\/?(.*)`)
		matches := regex.FindStringSubmatch(r.URL.Path)
		bucketName := matches[1]
		path := matches[2]

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
		if perPageStr := r.URL.Query().Get("perPage"); perPageStr != "" {
			if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
				perPage = pp
			}
		}

		// Get search parameter
		search := strings.TrimSpace(r.URL.Query().Get("search"))

		var objs []objectWithIcon
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

		// Filter objects based on search query
		if search != "" {
			searchLower := strings.ToLower(search)
			filteredObjs := make([]objectWithIcon, 0)
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
		sortObjects(objs, sortBy, sortOrder)

		// Calculate pagination
		totalItems := len(objs)
		totalPages := (totalItems + perPage - 1) / perPage
		if totalPages == 0 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}

		// Paginate objects
		start := (page - 1) * perPage
		end := start + perPage
		if start < 0 {
			start = 0
		}
		if end > totalItems {
			end = totalItems
		}
		if start < totalItems {
			objs = objs[start:end]
		} else {
			objs = []objectWithIcon{}
		}

		data := pageData{
			RootURL:      rootURL,
			BucketName:   bucketName,
			Objects:      objs,
			AllowDelete:  allowDelete,
			Paths:        removeEmptyStrings(strings.Split(path, "/")),
			CurrentPath:  path,
			CurrentS3:    nil,
			S3Instances:  nil,
			HasError:     false,
			ErrorMessage: "",
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
	result := make([]string, 0, len(input))
	for _, str := range input {
		if str == "" {
			continue
		}
		result = append(result, str)
	}
	return result
}

// sortObjects sorts the objects based on the specified field and order
func sortObjects(objs []objectWithIcon, sortBy, sortOrder string) {
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
