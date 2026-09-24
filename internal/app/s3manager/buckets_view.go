package s3manager

import (
	"fmt"
	"io/fs"
	"net/http"
	"slices"

	"github.com/minio/minio-go/v7"
)

// HandleBucketsView renders all buckets of an S3 instance on an HTML page.
func HandleBucketsView(instances S3Instances, templates fs.FS, opts Options) http.HandlerFunc {
	type pageData struct {
		RootURL      string
		Buckets      []minio.BucketInfo
		AllowDelete  bool
		CurrentS3    *S3Instance
		S3Instances  S3Instances
		HasError     bool
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
			AllowDelete: opts.AllowDelete,
			CurrentS3:   instance,
			S3Instances: instances,
		}

		buckets, err := instance.Client.ListBuckets(r.Context())
		switch {
		case err != nil && len(instance.Buckets) == 0:
			// An unreachable instance is reported on the page itself so that
			// the user can switch to another one instead of being stuck on an
			// error page.
			data.HasError = true
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

// filterBuckets narrows a bucket listing down to the single bucket the app is
// restricted to, if that bucket exists.
func filterBuckets(buckets []minio.BucketInfo, name string) []minio.BucketInfo {
	for _, bucket := range buckets {
		if bucket.Name == name {
			return []minio.BucketInfo{bucket}
		}
	}

	return nil
}
