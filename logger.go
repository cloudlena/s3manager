package main

import (
	"log"
	"net/http"
	"time"
)

// Logger logs HTTP requests
func Logger() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			defer func() {
				log.Printf(
					"%s\t%s\t%s",
					r.Method,
					r.RequestURI,
					time.Since(start),
				)
			}()

			next(w, r)
		}
	}
}
