package s3manager

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// publicAccessClient probes object URLs. An unresponsive endpoint must not
// block the request forever, hence a client with a timeout of its own.
var publicAccessClient = &http.Client{Timeout: 10 * time.Second}

// PublicObjectURL returns the link under which an object can be read without
// credentials, provided the bucket allows it. A configured PublicURL template
// has its {bucket} and {key} placeholders filled in; otherwise the object is
// addressed on the S3 endpoint the same way the client addresses its bucket.
func (instance *S3Instance) PublicObjectURL(bucketName, objectName string) string {
	key := escapeObjectKey(objectName)
	if instance.PublicURL != "" {
		return strings.NewReplacer("{bucket}", bucketName, "{key}", key).Replace(instance.PublicURL)
	}

	endpoint := *instance.Client.EndpointURL()
	if virtualHostStyle(endpoint, bucketName, instance.BucketLookup) {
		endpoint.Host = bucketName + "." + endpoint.Host
	} else {
		key = bucketName + "/" + key
	}

	return strings.TrimSuffix(endpoint.String(), "/") + "/" + key
}

// virtualHostStyle reports whether a bucket is addressed in the host name
// (bucket.endpoint) rather than in the path (endpoint/bucket), deciding
// `Auto` the same way minio-go does.
func virtualHostStyle(endpoint url.URL, bucketName string, lookup minio.BucketLookupType) bool {
	switch lookup {
	case minio.BucketLookupDNS:
		return true
	case minio.BucketLookupPath:
		return false
	default:
		return s3utils.IsVirtualHostSupported(endpoint, bucketName)
	}
}

// escapeObjectKey escapes an object key for use in a URL path, keeping the
// slashes that separate its segments.
func escapeObjectKey(objectName string) string {
	segments := strings.Split(objectName, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return strings.Join(segments, "/")
}

// HandleCheckPublicAccess returns an object's public link and checks whether
// the object is actually accessible under it.
func HandleCheckPublicAccess(instances S3Instances) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instance, ok := resolveInstance(w, r, instances)
		if !ok {
			return
		}

		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]

		// The bucket name may end up in the host name of the link, so it must
		// not be able to point the probe anywhere else.
		if err := s3utils.CheckValidBucketName(bucketName); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		publicURL := instance.PublicObjectURL(bucketName, objectName)

		req, err := http.NewRequestWithContext(r.Context(), http.MethodHead, publicURL, nil)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error creating request: %w", err))
			return
		}

		// An endpoint that cannot be reached at all answers the question too:
		// the object is not publicly accessible.
		statusCode := 0
		if resp, err := publicAccessClient.Do(req); err == nil {
			defer func() { _ = resp.Body.Close() }()
			statusCode = resp.StatusCode
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"url":        publicURL,
			"accessible": statusCode == http.StatusOK,
			"statusCode": statusCode,
		})
	}
}
