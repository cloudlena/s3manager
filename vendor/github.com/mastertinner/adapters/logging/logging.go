package logging

import (
	"log"
	"net/http"
	"time"

	"github.com/mastertinner/adapters"
)

// Handler logs HTTP requests.
func Handler(logger *log.Logger) adapters.Adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(start time.Time) {
				logger.Printf(
					"%s\t%s\t%s",
					r.Method,
					r.RequestURI,
					time.Since(start),
				)
			}(time.Now())

			next.ServeHTTP(w, r)
		})
	}
}
