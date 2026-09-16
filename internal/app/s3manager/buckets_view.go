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
		AllowDelete  bool
		CurrentS3    *S3Instance
		S3Instances  S3Instances
		UserName     string
		HasError     bool
		ErrorMessage string
	}

	renderer := newPageRenderer(templates, "buckets.html.tmpl")

	return func(w http.ResponseWriter, r *http.Request) {
		instance, ok := resolveInstance(w, r, instances)
		if !ok {
			return
		}

		opts := effectiveOptions(r.Context(), opts)
		data := pageData{
			RootURL:     opts.RootURL,
			AllowDelete: opts.AllowDelete,
			CurrentS3:   instance,
			S3Instances: instances,
			UserName:    userName(r.Context()),
		}

		buckets, err := instance.Client.ListBuckets(r.Context())
		switch {
		case err != nil:
			// An unreachable instance is reported on the page itself so that
			// the user can switch to another one instead of being stuck on an
			// error page.
			data.HasError = true
			data.ErrorMessage = fmt.Sprintf("Unable to connect to S3 instance '%s'. Please check the credentials and try switching to another instance.", instance.Name)
		case opts.BucketName != "":
			data.Buckets = filterBuckets(buckets, opts.BucketName)
		default:
			data.Buckets = buckets
		}

		renderer(w, data)
	}
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
