package s3manager

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

// Error messages an S3 client may report, used to map failures onto status codes.
const (
	msgBucketDoesNotExist = "The specified bucket does not exist"
	msgKeyDoesNotExist    = "The specified key does not exist"
)

// handleHTTPError responds with the error and a status code derived from it.
func handleHTTPError(w http.ResponseWriter, err error) {
	var syntaxErr *json.SyntaxError

	code := http.StatusInternalServerError
	switch {
	case errors.As(err, &syntaxErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		code = http.StatusUnprocessableEntity
	case strings.Contains(err.Error(), msgBucketDoesNotExist), strings.Contains(err.Error(), msgKeyDoesNotExist):
		code = http.StatusNotFound
	}

	http.Error(w, err.Error(), code)

	if code >= http.StatusInternalServerError {
		log.Println(err)
	}
}

// writeJSON responds with the JSON encoding of body. The status code is already
// written when encoding fails, so such an error can only be logged.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}
