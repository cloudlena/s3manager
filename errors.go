package s3manager

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

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
	var code int
	switch err := errors.Cause(err).(type) {
	case *json.SyntaxError:
		code = http.StatusUnprocessableEntity
	default:
		if err == io.EOF {
			code = http.StatusUnprocessableEntity
		} else if err == ErrBucketDoesNotExist ||
			err == ErrKeyDoesNotExist {
			code = http.StatusNotFound
		} else {
			code = http.StatusInternalServerError
		}
	}

	http.Error(w, http.StatusText(code), code)

	// Log if server error
	if code >= 500 {
		log.Println(err)
	}
}
