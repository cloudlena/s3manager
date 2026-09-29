package s3manager

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"slices"

	"github.com/minio/minio-go/v7"
)

// handleHTTPError responds with the error and a status code derived from it.
func handleHTTPError(w http.ResponseWriter, err error) {
	var syntaxErr *json.SyntaxError

	code := http.StatusInternalServerError
	switch {
	case errors.As(err, &syntaxErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		code = http.StatusUnprocessableEntity
	case hasS3ErrorCode(err, minio.NoSuchBucket, minio.NoSuchKey):
		code = http.StatusNotFound
	}

	http.Error(w, err.Error(), code)

	if code >= http.StatusInternalServerError {
		log.Println(err)
	}
}

// hasS3ErrorCode reports whether err is an S3 error response with one of the
// given codes.
func hasS3ErrorCode(err error, codes ...string) bool {
	var errResp minio.ErrorResponse
	return errors.As(err, &errResp) && slices.Contains(codes, errResp.Code)
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
