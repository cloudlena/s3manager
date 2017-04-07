package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexViewHandler(t *testing.T) {
	assert := assert.New(t)

	tests := map[string]struct {
		expectedStatusCode   int
		expectedBodyContains string
	}{
		"success": {
			expectedStatusCode:   http.StatusPermanentRedirect,
			expectedBodyContains: "Redirect",
		},
	}

	for tcID, tc := range tests {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		assert.NoError(err, tcID)

		rr := httptest.NewRecorder()
		handler := IndexViewHandler()

		handler.ServeHTTP(rr, req)

		assert.Equal(tc.expectedStatusCode, rr.Code, tcID)
		assert.Contains(rr.Body.String(), tc.expectedBodyContains, tcID)
	}
}
