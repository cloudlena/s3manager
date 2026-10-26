package s3manager

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
)

// renderableContentTypes are the content types a browser can display without
// being able to run scripts in the app's origin.
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

// textualContentTypes are the content types that aren't safe to render (most
// notably HTML, SVG and XML, which can carry scripts or stylesheets) but are
// still readable as plain text.
var textualContentTypes = []string{
	"application/javascript",
	"application/toml",
	"application/x-sh",
	"application/x-yaml",
	"application/xml",
	"application/yaml",
	"text/",
}

// inlineContentType maps the content type stored in S3 to the one used when
// displaying an object in the browser. Textual objects that the browser can't
// render safely are shown as plain text so that opening an object can never
// turn into stored XSS. Any other object, such as a spreadsheet or an archive,
// can't be displayed at all, which is reported by returning false.
func inlineContentType(contentType string) (string, bool) {
	base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if matchesContentType(base, renderableContentTypes) {
		return contentType, true
	}
	if matchesContentType(base, textualContentTypes) || strings.HasSuffix(base, "+xml") || strings.HasSuffix(base, "+json") {
		return "text/plain; charset=utf-8", true
	}

	return "", false
}

// matchesContentType reports whether a lowercase base content type is one of
// the given ones, where an entry ending in a slash stands for its whole type.
func matchesContentType(base string, contentTypes []string) bool {
	for _, ct := range contentTypes {
		if base == ct || (strings.HasSuffix(ct, "/") && strings.HasPrefix(base, ct)) {
			return true
		}
	}

	return false
}

// HandleGetObject serves an object to the client. It is downloaded as an
// attachment unless the inline query parameter is set, in which case it is
// served for display in the browser if the browser can display it.
func HandleGetObject(s3 S3, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]

		object, err := s3.GetObject(r.Context(), bucketName, objectName, minio.GetObjectOptions{
			VersionID:            requestedVersion(r, opts),
			ServerSideEncryption: opts.SSE,
		})
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting object: %w", err))
			return
		}
		defer func() { _ = object.Close() }()

		// GetObject is lazy and only Stat sends the request, so an object that
		// can't be read is reported before any of the response is written.
		info, err := object.Stat()
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error getting object info: %w", err))
			return
		}

		// The content type has to come from S3 rather than from sniffing the
		// body, which could turn an HTML object into stored XSS.
		disposition, contentType := "attachment", "application/octet-stream"
		if r.URL.Query().Get("inline") == "true" {
			if inlineType, ok := inlineContentType(info.ContentType); ok {
				disposition, contentType = "inline", inlineType
			}
		}
		w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": path.Base(objectName)}))
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// The response is already under way once copying starts, so an error
		// can only be logged.
		if _, err := io.Copy(w, object); err != nil {
			log.Printf("error copying object %s to response writer: %v", objectName, err)
		}
	}
}

// requestedVersion returns the object version a request asks for. It is
// ignored unless the versions feature is enabled, so disabling SHOW_VERSIONS
// also prevents access to old versions.
func requestedVersion(r *http.Request, opts Options) string {
	if !opts.ShowVersions {
		return ""
	}

	return r.URL.Query().Get("versionId")
}
