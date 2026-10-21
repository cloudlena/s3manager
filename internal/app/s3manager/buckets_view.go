package s3manager

import (
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"github.com/minio/minio-go/v7"
)

// HandleBucketsView renders all buckets of an S3 instance on an HTML page.
func HandleBucketsView(instances S3Instances, templates fs.FS, opts Options) http.HandlerFunc {
	type pageData struct {
		RootURL      string
		Buckets      []minio.BucketInfo
		CurrentS3    *S3Instance
		S3Instances  S3Instances
		ErrorMessage string
	}

	renderer := newPageRenderer(templates, "buckets.html.tmpl")

	return func(w http.ResponseWriter, r *http.Request) {
		instance, ok := resolveInstance(w, r, instances)
		if !ok {
			return
		}

		data := pageData{
			RootURL:     opts.RootURL,
			CurrentS3:   instance,
			S3Instances: instances,
		}

		buckets, err := instance.Client.ListBuckets(r.Context())
		switch {
		case err != nil && len(instance.Buckets) == 0:
			// An unreachable instance is reported on the page itself so that
			// the user can switch to another one instead of being stuck on an
			// error page.
			data.ErrorMessage = fmt.Sprintf("Unable to connect to S3 instance '%s'. Please check the credentials and try switching to another instance.", instance.Name)
		case opts.BucketName != "":
			data.Buckets = filterBuckets(addConfiguredBuckets(buckets, instance.Buckets), opts.BucketName)
		default:
			// Listing buckets is refused for anonymous access, so the
			// configured buckets stand on their own when it fails.
			data.Buckets = addConfiguredBuckets(buckets, instance.Buckets)
		}

		renderer(w, data)
	}
}

// addConfiguredBuckets appends the configured bucket names that a listing
// does not already contain. Their creation date is unknown and stays zero.
func addConfiguredBuckets(buckets []minio.BucketInfo, names []string) []minio.BucketInfo {
	for _, name := range names {
		if !slices.ContainsFunc(buckets, func(bucket minio.BucketInfo) bool { return bucket.Name == name }) {
			buckets = append(buckets, minio.BucketInfo{Name: name})
		}
	}

	return buckets
}

// filterBuckets narrows a bucket listing down to the buckets the app is
// restricted to. allowed is a single bucket name, or a comma-separated list
// of names for an app restricted to more than one bucket. The result keeps
// the order ListBuckets returned, not the order in allowed.
func filterBuckets(buckets []minio.BucketInfo, allowed string) []minio.BucketInfo {
	names := strings.Split(allowed, ",")
	for i, name := range names {
		names[i] = strings.TrimSpace(name)
	}

	var filtered []minio.BucketInfo

	for _, bucket := range buckets {
		if slices.Contains(names, bucket.Name) {
			filtered = append(filtered, bucket)
		}
	}

	return filtered
}
