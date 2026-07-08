package s3manager

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

const defaultPerPage = 25

// objectWithIcon represents an S3 object with additional display properties
type objectWithIcon struct {
	Key              string
	Size             int64
	SizeDisplay      string
	LastModified     time.Time
	Owner            string
	Icon             string
	IsFolder         bool
	DisplayName      string
	VersionID        string
	IsLatest         bool
	IsDeleteMarker   bool
	VersionCount     int
	GroupIndex       int
	IsPrimaryVersion bool
}

// annotateVersionGroups sets VersionCount, GroupIndex and IsPrimaryVersion on
// each object so the template can collapse older versions under their latest
// version by default. IsPrimaryVersion picks exactly one visible row per key:
// the one the provider marked IsLatest, or (since some S3-compatible providers
// leave IsLatest unset — notably folder entries synthesized from
// CommonPrefixes, which are never version-aware) the first entry seen for that
// key. Relying on the raw IsLatest flag alone would hide every row in a group
// where no entry has it set, making the bucket appear empty.
func annotateVersionGroups(objs []objectWithIcon) {
	counts := make(map[string]int, len(objs))
	groupIndex := make(map[string]int, len(objs))
	primaryIndex := make(map[string]int, len(objs))
	nextIndex := 0

	for i, obj := range objs {
		counts[obj.Key]++
		if _, ok := groupIndex[obj.Key]; !ok {
			groupIndex[obj.Key] = nextIndex
			nextIndex++
			primaryIndex[obj.Key] = i
		} else if obj.IsLatest {
			primaryIndex[obj.Key] = i
		}
	}

	for i := range objs {
		key := objs[i].Key
		objs[i].VersionCount = counts[key]
		objs[i].GroupIndex = groupIndex[key]
		objs[i].IsPrimaryVersion = primaryIndex[key] == i
	}
}

// listObjectsOptions builds the minio.ListObjectsOptions used to list a bucket's objects.
func listObjectsOptions(listRecursive, showVersions bool, prefix string) minio.ListObjectsOptions {
	return minio.ListObjectsOptions{
		Recursive:    listRecursive,
		Prefix:       prefix,
		WithVersions: showVersions,
	}
}

// listObjectsForBucketView lists a bucket's objects, converting each minio.ObjectInfo
// into an objectWithIcon. If showVersions is set but the versioned listing fails, or
// comes back empty (some S3-compatible providers don't support listing object
// versions and either reject the request outright or silently return nothing
// instead of erroring), it transparently falls back to a normal listing so the
// bucket can still be browsed. The returned bool reports whether version
// information is actually present in the result.
func listObjectsForBucketView(ctx context.Context, s3 S3, bucketName, path string, listRecursive, showVersions bool) ([]objectWithIcon, bool, error) {
	if !showVersions {
		objs, err := collectObjects(ctx, s3, bucketName, path, listObjectsOptions(listRecursive, false, path))
		return objs, false, err
	}

	objs, err := collectObjects(ctx, s3, bucketName, path, listObjectsOptions(listRecursive, true, path))
	if err == nil && len(objs) > 0 {
		return objs, true, nil
	}

	fallbackObjs, fallbackErr := collectObjects(ctx, s3, bucketName, path, listObjectsOptions(listRecursive, false, path))
	if fallbackErr != nil {
		return nil, false, fallbackErr
	}
	return fallbackObjs, false, nil
}

// collectObjects drains an S3 ListObjects channel into a slice, returning the
// first error encountered (if any) instead of a partial, half-listed result.
func collectObjects(ctx context.Context, s3 S3, bucketName, path string, opts minio.ListObjectsOptions) ([]objectWithIcon, error) {
	var objs []objectWithIcon
	objectCh := s3.ListObjects(ctx, bucketName, opts)
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objs = append(objs, toObjectWithIcon(object, path))
	}
	return objs, nil
}

// friendlyListObjectsErrorMessage turns a raw S3 listing error into an
// actionable, user-facing message for the bucket view's error banner.
func friendlyListObjectsErrorMessage(err error, bucketName, instanceName string) string {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "AccessDenied") || strings.Contains(msg, "InvalidAccessKeyId") || strings.Contains(msg, "SignatureDoesNotMatch"):
		return fmt.Sprintf("Unable to access bucket '%s' on S3 instance '%s'. Please check the credentials and try switching to another instance.", bucketName, instanceName)
	case strings.Contains(msg, ErrBucketDoesNotExist):
		return fmt.Sprintf("Bucket '%s' does not exist on S3 instance '%s'. Please try switching to another instance or go back to the buckets list.", bucketName, instanceName)
	default:
		return fmt.Sprintf("Unable to list objects in bucket '%s' on S3 instance '%s': %s", bucketName, instanceName, msg)
	}
}

// toObjectWithIcon converts a minio.ObjectInfo into the template-facing objectWithIcon.
func toObjectWithIcon(object minio.ObjectInfo, path string) objectWithIcon {
	return objectWithIcon{
		Key:            object.Key,
		Size:           object.Size,
		SizeDisplay:    FormatFileSize(object.Size),
		LastModified:   object.LastModified,
		Owner:          object.Owner.DisplayName,
		Icon:           icon(object.Key),
		IsFolder:       strings.HasSuffix(object.Key, "/"),
		DisplayName:    strings.TrimSuffix(strings.TrimPrefix(object.Key, path), "/"),
		VersionID:      object.VersionID,
		IsLatest:       object.IsLatest,
		IsDeleteMarker: object.IsDeleteMarker,
	}
}

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(s3 S3, templates fs.FS, allowDelete bool, listRecursive bool, rootURL string, showVersions bool) http.HandlerFunc {
	type pageData struct {
		RootURL             string
		BucketName          string
		Objects             []objectWithIcon
		AllowDelete         bool
		Paths               []string
		CurrentPath         string
		Endpoint            string
		CurrentS3           *S3Instance
		S3Instances         []*S3Instance
		HasError            bool
		ErrorMessage        string
		SortBy              string
		SortOrder           string
		Page                int
		PerPage             int
		TotalItems          int
		TotalPages          int
		HasPrevPage         bool
		HasNextPage         bool
		Search              string
		ShowVersions        bool
		VersionsUnavailable bool
	}

	return func(w http.ResponseWriter, r *http.Request) {
		regex := regexp.MustCompile(`\/buckets\/([^\/]*)\/?(.*)`)
		matches := regex.FindStringSubmatch(r.URL.Path)
		bucketName := matches[1]
		path, rqerr := url.QueryUnescape(matches[2])
		if rqerr != nil {
			handleHTTPError(w, rqerr)
			return
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

		perPage := defaultPerPage
		if perPageStr := r.URL.Query().Get("perPage"); perPageStr != "" {
			if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
				perPage = pp
			}
		}

		// Get search parameter
		search := strings.TrimSpace(r.URL.Query().Get("search"))

		objs, versionsShown, err := listObjectsForBucketView(r.Context(), s3, bucketName, path, listRecursive, showVersions)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error listing objects: %w", err))
			return
		}

		if versionsShown {
			annotateVersionGroups(objs)
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
			RootURL:             rootURL,
			BucketName:          bucketName,
			Objects:             objs,
			AllowDelete:         allowDelete,
			Paths:               removeEmptyStrings(strings.Split(path, "/")),
			CurrentPath:         path,
			Endpoint:            s3.EndpointURL().String(),
			CurrentS3:           nil,
			S3Instances:         nil,
			HasError:            false,
			ErrorMessage:        "",
			SortBy:              sortBy,
			SortOrder:           sortOrder,
			Page:                page,
			PerPage:             perPage,
			TotalItems:          totalItems,
			TotalPages:          totalPages,
			HasPrevPage:         page > 1,
			HasNextPage:         page < totalPages,
			Search:              search,
			ShowVersions:        versionsShown,
			VersionsUnavailable: showVersions && !versionsShown,
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
