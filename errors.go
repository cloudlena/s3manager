package s3manager

import (
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Error codes commonly used throughout the application
const (
	errDecodingBody      = "error decoding body JSON"
	errEncodingJSON      = "error encoding JSON"
	errExecutingTemplate = "error executing template"
	errParsingForm       = "error parsing form"
	errParsingTemplates  = "error parsing template files"
)

// Errors that may be returned from an S3 client.
var (
	ErrBucketDoesNotExist = errors.New("The specified bucket does not exist.")
	ErrKeyDoesNotExist    = errors.New("The specified key does not exist.")
)

// handleHTTPError handles HTTP errors.
func handleHTTPError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if errors.Cause(err) == ErrBucketDoesNotExist {
		code = http.StatusNotFound
	} else if errors.Cause(err) == ErrKeyDoesNotExist {
		code = http.StatusNotFound
	} else if strings.Contains(err.Error(), errDecodingBody) {
		code = http.StatusUnprocessableEntity
	} else if strings.Contains(err.Error(), errParsingForm) {
		code = http.StatusUnprocessableEntity
	}

	http.Error(w, http.StatusText(code), code)

	// Log if server error
	if code >= 500 {
		log.Println(err)
	}
}
