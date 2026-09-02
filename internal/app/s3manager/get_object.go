package s3manager

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// renderableContentTypes are the content types a browser can display without
// being able to run scripts in the app's origin. Anything else (most notably
// HTML, SVG and XML, which can carry scripts or stylesheets) is served as plain
// text so that opening an object can never turn into stored XSS.
var renderableContentTypes = []string{
	"application/json",
	"application/pdf",
	"audio/",
	"image/bmp",
	"image/avif",
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/tiff",
	"image/webp",
	"text/csv",
	"text/markdown",
	"text/plain",
	"video/",
}

// inlineContentType maps the content type stored in S3 to the one used when
// displaying an object in the browser.
func inlineContentType(contentType string) string {
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	for _, renderable := range renderableContentTypes {
		if base == renderable || (strings.HasSuffix(renderable, "/") && strings.HasPrefix(base, renderable)) {
			return contentType
		}
	}

	return "text/plain; charset=utf-8"
}

// HandleGetObject serves an object to the client. It is downloaded as an
// attachment unless the inline query parameter is set, in which case it is
// served for display in the browser.
func HandleGetObject(s3 S3, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]
		// Ignore versionId unless the versions feature is enabled, so
		// disabling SHOW_VERSIONS also prevents access to old versions.
		versionID := ""
		if opts.ShowVersions {
			versionID = r.URL.Query().Get("versionId")
		}
		inline := r.URL.Query().Get("inline") == "true"

		contentType := ""
		if inline {
			// The content type has to come from S3 rather than from sniffing
			// the body, so it is looked up before streaming the object.
			info, err := s3.StatObject(r.Context(), bucketName, objectName, minio.StatObjectOptions{VersionID: versionID})
			if err != nil {
				handleHTTPError(w, fmt.Errorf("error getting object metadata: %w", err))
				return
			}
			contentType = inlineContentType(info.ContentType)
		}

		object, err := s3.GetObject(r.Context(), bucketName, objectName, minio.GetObjectOptions{VersionID: versionID})
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting object: %w", err))
			return
		}

		switch {
		case inline:
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", path.Base(objectName)))
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("X-Content-Type-Options", "nosniff")
		case opts.ForceDownload:
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", objectName))
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		_, err = io.Copy(w, object)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error copying object to response writer: %w", err))
			return
		}
	}
}
