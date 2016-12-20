package main

import "net/http"

// Route represents a path of the API
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is an array of routes
type Routes []Route

var routes = Routes{
	Route{
		"Redirect to /buckets",
		"GET",
		"/",
		indexPageHandler,
	},
	Route{
		"Load Buckets Page",
		"GET",
		"/buckets",
		bucketsPageHandler,
	},
	Route{
		"Load Bucket Page",
		"GET",
		"/buckets/{bucketID}",
		bucketPageHandler,
	},
}
