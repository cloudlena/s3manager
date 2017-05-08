package adapters

import "net/http"

// Adapter is an HTTP middleware.
type Adapter func(http.Handler) http.Handler
