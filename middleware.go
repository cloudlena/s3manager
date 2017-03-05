package main

import "net/http"

// Middleware is an HTTP middleware
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain applies middleware to an HTTP handler function
func Chain(f http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		f = middlewares[i](f)
	}
	return f
}
