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
		"/buckets/{bucketName}",
		bucketPageHandler,
	},
	Route{
		"Create Bucket",
		"POST",
		"/api/buckets",
		createBucketHandler,
	},
	Route{
		"Download Object",
		"GET",
		"/api/buckets/{bucketName}/objects/{objectName}",
		getObjectHandler,
	},
	Route{
		"Upload Object",
		"POST",
		"/api/buckets/{bucketName}/objects",
		createObjectHandler,
	},
	Route{
		"Delete Object",
		"DELETE",
		"/api/buckets/{bucketName}/objects/{objectName}",
		deleteObjectHandler,
	},
}
