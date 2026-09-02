package s3manager

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const defaultPerPage = 25

// bucketPathPattern matches the bucket name and the path within the bucket of a
// bucket view URL, which has the form /{instance}/buckets/{bucket}/{path...}.
var bucketPathPattern = regexp.MustCompile(`/[^/]+/buckets/([^/]*)/?(.*)`)

// listingQuery holds the sorting, pagination and search parameters of a bucket
// view request.
type listingQuery struct {
	SortBy    string
	SortOrder string
	Page      int
	PerPage   int
	ShowAll   bool
	Search    string
}

// HandleBucketView shows the details page of a bucket.
func HandleBucketView(instances S3Instances, templates fs.FS, opts Options) http.HandlerFunc {
	type pageData struct {
		objectPage
		RootURL             string
		BucketName          string
		CurrentPath         string
		Paths               []string
		Endpoint            string
		AllowDelete         bool
		CurrentS3           *S3Instance
		S3Instances         S3Instances
		HasError            bool
		ErrorMessage        string
		SortBy              string
		SortOrder           string
		Search              string
		ShowVersions        bool
		VersionsUnavailable bool
		ShowMetadata        bool
	}

	renderer := newPageRenderer(templates, "bucket.html.tmpl")

	return func(w http.ResponseWriter, r *http.Request) {
		instance, ok := resolveInstance(w, r, instances)
		if !ok {
			return
		}

		bucketName, path, err := parseBucketPath(r.URL.Path)
		if err != nil {
			handleHTTPError(w, err)
			return
		}

		query := parseListingQuery(r.URL.Query())
		data := pageData{
			RootURL:      opts.RootURL,
			BucketName:   bucketName,
			CurrentPath:  path,
			Paths:        removeEmptyStrings(strings.Split(path, "/")),
			Endpoint:     instance.Client.EndpointURL().String(),
			AllowDelete:  opts.AllowDelete,
			CurrentS3:    instance,
			S3Instances:  instances,
			SortBy:       query.SortBy,
			SortOrder:    query.SortOrder,
			Search:       query.Search,
			ShowMetadata: opts.ShowMetadata,
		}

		objs, versionsShown, err := listObjects(r.Context(), instance.Client, bucketName, path, opts.ListRecursive, opts.ShowVersions)
		if err != nil {
			// A failed listing is reported on the page itself so that the user
			// can switch instances or go back instead of being stuck on an
			// error page.
			data.HasError = true
			data.ErrorMessage = listObjectsErrorMessage(err, bucketName, instance.Name)
			renderer(w, data)
			return
		}

		if versionsShown {
			annotateVersionGroups(objs)
		}
		if query.Search != "" {
			objs = filterObjects(objs, query.Search)
		}

		data.objectPage = paginateObjects(objs, query, versionsShown)
		data.ShowVersions = versionsShown
		// Only warn about unavailable versions when there is content to show;
		// an empty bucket legitimately produces an empty versioned listing.
		data.VersionsUnavailable = opts.ShowVersions && !versionsShown && len(objs) > 0

		renderer(w, data)
	}
}

// parseBucketPath extracts the bucket name and the path within the bucket from
// a bucket view URL.
func parseBucketPath(urlPath string) (string, string, error) {
	matches := bucketPathPattern.FindStringSubmatch(urlPath)
	if matches == nil {
		return "", "", fmt.Errorf("invalid bucket path: %s", urlPath)
	}

	return matches[1], matches[2], nil
}

// parseListingQuery reads a bucket view request's query parameters, falling
// back to defaults for missing or invalid values.
func parseListingQuery(params url.Values) listingQuery {
	query := listingQuery{
		SortBy:    params.Get("sortBy"),
		SortOrder: params.Get("sortOrder"),
		Page:      1,
		PerPage:   defaultPerPage,
		Search:    strings.TrimSpace(params.Get("search")),
	}

	if query.SortBy == "" {
		query.SortBy = "key"
	}
	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}
	if page, err := strconv.Atoi(params.Get("page")); err == nil && page > 0 {
		query.Page = page
	}
	if perPage, err := strconv.Atoi(params.Get("perPage")); err == nil {
		switch {
		case perPage > 0:
			query.PerPage = perPage
		case perPage == 0 || perPage == -1:
			query.ShowAll = true
		}
	}

	return query
}

// listObjectsErrorMessage turns a raw S3 listing error into an actionable,
// user-facing message for the bucket view's error banner.
func listObjectsErrorMessage(err error, bucketName, instanceName string) string {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "AccessDenied"), strings.Contains(msg, "InvalidAccessKeyId"), strings.Contains(msg, "SignatureDoesNotMatch"):
		return fmt.Sprintf("Unable to access bucket '%s' on S3 instance '%s'. Please check the credentials and try switching to another instance.", bucketName, instanceName)
	case strings.Contains(msg, msgBucketDoesNotExist):
		return fmt.Sprintf("Bucket '%s' does not exist on S3 instance '%s'. Please try switching to another instance or go back to the buckets list.", bucketName, instanceName)
	default:
		return fmt.Sprintf("Unable to list objects in bucket '%s' on S3 instance '%s': %s", bucketName, instanceName, msg)
	}
}

// removeEmptyStrings drops the empty segments of a split path.
func removeEmptyStrings(input []string) []string {
	result := make([]string, 0, len(input))
	for _, str := range input {
		if str != "" {
			result = append(result, str)
		}
	}

	return result
}
