package s3manager

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/pkg/errors"
)

// Error codes that may be returned from an S3 client.
const (
	ErrBucketDoesNotExist = "The specified bucket does not exist"
	ErrKeyDoesNotExist    = "The specified key does not exist"
)

// handleHTTPError handles HTTP errors.
func handleHTTPError(w http.ResponseWriter, err error) {
	cause := errors.Cause(err)
	code := http.StatusInternalServerError

	_, ok := cause.(*json.SyntaxError)
	if ok {
		code = http.StatusUnprocessableEntity
	}
	switch {
	case cause == io.EOF || cause == io.ErrUnexpectedEOF:
		code = http.StatusUnprocessableEntity
	case cause.Error() == ErrBucketDoesNotExist || cause.Error() == ErrKeyDoesNotExist:
		code = http.StatusNotFound
	}

	http.Error(w, http.StatusText(code), code)

	// Log if server error
	if code >= 500 {
		log.Println(err)
	}
}
