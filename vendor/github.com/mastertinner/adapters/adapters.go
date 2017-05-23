// Package adapters is a very lightweight framework for HTTP middleware.
package adapters

import "net/http"

// Adapter is an HTTP middleware.
type Adapter func(http.Handler) http.Handler

// Adapt adds adapters to an HTTP handler.
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for i := len(adapters) - 1; i >= 0; i-- {
		h = adapters[i](h)
	}

	return h
}
