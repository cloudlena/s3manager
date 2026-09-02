package s3manager

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// maxPresignedURLExpiry is the longest lifetime S3 accepts for a presigned URL.
const maxPresignedURLExpiry = 7 * 24 * time.Hour

// HandleGenerateURL generates a presigned URL.
func HandleGenerateURL(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]

		seconds, err := strconv.ParseInt(r.URL.Query().Get("expiry"), 10, 64)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error converting expiry: %w", err))
			return
		}

		if seconds < 1 || seconds > int64(maxPresignedURLExpiry/time.Second) {
			handleHTTPError(w, fmt.Errorf("invalid expiry value: %v", seconds))
			return
		}

		expiry := time.Duration(seconds) * time.Second
		presignedURL, err := s3.PresignedGetObject(r.Context(), bucketName, objectName, expiry, url.Values{})
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error generating url: %w", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"url": presignedURL.String()})
	}
}
