package s3manager

import (
	"fmt"
	"io/fs"
	"net/http"

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
