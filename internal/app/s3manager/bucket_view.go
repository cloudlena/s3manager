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
	// Cursors is the trail of the cursors of the preceding pages, one per page,
	// as collected by the pager of a cursor-paged view. Keeping the whole trail
	// in the URL rather than just the current cursor is what lets such a view
	// still number its pages and step back to the previous one.
	Cursors []string
}

// Cursor is the key the current page's listing resumes after, empty on the first
// page.
func (q listingQuery) Cursor() string {
	if len(q.Cursors) == 0 {
		return ""
	}

	return q.Cursors[len(q.Cursors)-1]
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
		UserName            string
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
		Truncated           bool
		MaxScanObjects      int
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

		opts := effectiveOptions(r.Context(), opts)
		query := parseListingQuery(r.URL.Query())
		data := pageData{
			RootURL:        opts.RootURL,
			BucketName:     bucketName,
			CurrentPath:    path,
			Paths:          removeEmptyStrings(strings.Split(path, "/")),
			Endpoint:       instance.Client.EndpointURL().String(),
			AllowDelete:    opts.AllowDelete,
			UserName:       userName(r.Context()),
			CurrentS3:      instance,
			S3Instances:    instances,
			SortBy:         query.SortBy,
			SortOrder:      query.SortOrder,
			Search:         query.Search,
			ShowMetadata:   opts.ShowMetadata,
			MaxScanObjects: maxScanObjects,
		}

		// The default view is served one page at a time straight from S3, which
		// keeps its cost independent of how many objects the bucket holds. Every
		// other view needs the whole prefix in memory to do its job.
		cursorPaging := cursorPagingPossible(query, opts)

		var listing objectListing
		if cursorPaging {
			listing, err = listObjectPage(r.Context(), instance.Client, bucketName, path, query.Cursor(), opts.ListRecursive, query.PerPage)
		} else {
			listing, err = listAllObjects(r.Context(), instance.Client, bucketName, path, opts.ListRecursive, opts.ShowVersions)
		}
		if err != nil {
			// A failed listing is reported on the page itself so that the user
			// can switch instances or go back instead of being stuck on an
			// error page.
			data.HasError = true
			data.ErrorMessage = listObjectsErrorMessage(err, bucketName, instance.Name)
			renderer(w, data)
			return
		}

		if cursorPaging {
			data.objectPage = cursorPage(listing, query)
		} else {
			if listing.VersionsShown {
				annotateVersionGroups(listing.Objects)
			}
			if query.Search != "" {
				listing.Objects = filterObjects(listing.Objects, query.Search)
			}
			data.objectPage = paginateObjects(listing.Objects, query, listing.VersionsShown)
		}

		data.ShowVersions = listing.VersionsShown
		data.Truncated = listing.Truncated
		// Only warn about unavailable versions when there is content to show;
		// an empty bucket legitimately produces an empty versioned listing.
		data.VersionsUnavailable = opts.ShowVersions && !listing.VersionsShown && len(listing.Objects) > 0

		renderer(w, data)
	}
}

// cursorPagingPossible reports whether a request can be served by listing only
// the page it shows. S3 lists keys in ascending lexicographic order and can
// resume after a given key, but it cannot sort by anything else, cannot list
// backwards and has no search — so sorting by another column, sorting
// descending, searching and asking for every object at once all still need the
// full listing. So does a versioned listing, whose version groups have to be
// assembled before they can be split into pages.
func cursorPagingPossible(query listingQuery, opts Options) bool {
	return !opts.ShowVersions &&
		!query.ShowAll &&
		query.Search == "" &&
		query.SortBy == "key" &&
		query.SortOrder == "asc"
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
		Cursors:   removeEmptyStrings(params["cursor"]),
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
