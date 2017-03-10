package main

import (
	"log"
	"net/http"
	"time"
)

// Logger logs HTTP requests
func Logger() Adapter {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			defer func() {
				log.Printf(
					"%s\t%s\t%s",
					r.Method,
					r.RequestURI,
					time.Since(start),
				)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
