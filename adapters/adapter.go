package adapters

import "net/http"

// Adapter is an HTTP middleware
type Adapter func(http.Handler) http.Handler

// Adapt applies adapters to an HTTP handler function
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for i := len(adapters) - 1; i >= 0; i-- {
		h = adapters[i](h)
	}

	return h
}
