package s3manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// HandleCheckPublicAccess checks if an object is publicly accessible.
func HandleCheckPublicAccess(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]

		endpoint := s3.EndpointURL().String()
		if !strings.HasSuffix(endpoint, "/") {
			endpoint += "/"
		}
		
		// Construct the public URL
		// Note: This assumes path-style access (http://endpoint/bucket/object)
		// which is typical for MinIO and generic S3. 
		url := fmt.Sprintf("%s%s/%s", endpoint, bucketName, objectName)

		// Perform a HEAD request to check accessibility without downloading content
		resp, err := http.Head(url)
		isAccessible := false
		statusCode := 0

		if err != nil {
			// If we can't reach it, it's definitely not accessible or there's a network issue
			// We treat this as not accessible for the user's purpose
			isAccessible = false
		} else {
			defer resp.Body.Close()
			statusCode = resp.StatusCode
			// 200 OK means accessible. 
			// We might also consider 304 Not Modified as accessible if that ever happens on a fresh HEAD.
			isAccessible = resp.StatusCode == http.StatusOK
		}

		response := map[string]interface{}{
			"accessible": isAccessible,
			"statusCode": statusCode,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			handleHTTPError(w, fmt.Errorf("error encoding JSON: %w", err))
			return
		}
	}
}
