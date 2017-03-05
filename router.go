package main

import (
	"github.com/gorilla/mux"
)

// NewRouter creates a new router
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			HandlerFunc(route.HandlerFunc)
	}

	return router
}
