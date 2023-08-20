package s3manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func HandleGenerateUrl(s3 S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bucketName := mux.Vars(r)["bucketName"]
		objectName := mux.Vars(r)["objectName"]
		expiry := r.URL.Query().Get("expiry")

		parsedExpiry, err := strconv.ParseInt(expiry, 10, 0)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error when converting expiry: %w", err))
			return
		}

		if parsedExpiry > 7*24*60*60 || parsedExpiry < 1 {
			handleHTTPError(w, fmt.Errorf("invalid expiry value: %v", parsedExpiry))
			return
		}

		expirySecond := time.Duration(parsedExpiry * 1e9)
		reqParams := make(url.Values)
		url, err := s3.PresignedGetObject(r.Context(), bucketName, objectName, expirySecond, reqParams)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error when generate url: %w", err))
			return
		}

		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		err = encoder.Encode(map[string]string{"url": url.String()})
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error encoding JSON: %w", err))
			return
		}
	}
}
