package s3manager

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// publicAccessClient probes object URLs. An unresponsive endpoint must not
// block the request forever, hence a client with a timeout of its own.
var publicAccessClient = &http.Client{Timeout: 10 * time.Second}

// HandleCheckPublicAccess checks if an object is publicly accessible.
func HandleCheckPublicAccess(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]

		// Objects are addressed path-style (http://endpoint/bucket/object).
		endpoint := strings.TrimSuffix(s3.EndpointURL().String(), "/")
		publicURL := fmt.Sprintf("%s/%s/%s", endpoint, bucketName, objectName)

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
			"accessible": statusCode == http.StatusOK,
			"statusCode": statusCode,
		})
	}
}
