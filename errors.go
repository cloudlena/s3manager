package s3manager

import (
	"log"
	"net/http"
)

// Error codes that may be returned from an S3 backend.
const (
	ErrBucketDoesNotExist = "The specified bucket does not exist."
	ErrKeyDoesNotExist    = "The specified key does not exist."
)

// handleHTTPError handles HTTP errors.
func handleHTTPError(w http.ResponseWriter, statusCode int, err error) {
	msg := http.StatusText(statusCode)
	http.Error(w, msg, statusCode)

	logMsg := msg
	if err != nil {
		logMsg = logMsg + ": " + err.Error()
	}
	log.Println(logMsg)
}
