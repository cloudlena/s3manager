package main

import "net/http"

// Route represents a path of the API
type Route struct {
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is an array of routes
type Routes []Route

var routes = Routes{
	Route{
		"GET",
		"/",
		Chain(indexPageHandler, Logger()),
	},
	Route{
		"GET",
		"/buckets",
		Chain(bucketsPageHandler, Logger()),
	},
	Route{
		"GET",
		"/buckets/{bucketName}",
		Chain(bucketPageHandler, Logger()),
	},
	Route{
		"POST",
		"/api/buckets",
		Chain(createBucketHandler, Logger()),
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}",
		Chain(deleteBucketHandler, Logger()),
	},
	Route{
		"GET",
		"/api/buckets/{bucketName}/objects/{objectName}",
		Chain(getObjectHandler, Logger()),
	},
	Route{
		"POST",
		"/api/buckets/{bucketName}/objects",
		Chain(createObjectHandler, Logger()),
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}/objects/{objectName}",
		Chain(deleteObjectHandler, Logger()),
	},
}
