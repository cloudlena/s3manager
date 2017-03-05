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
		indexPageHandler,
	},
	Route{
		"GET",
		"/buckets",
		bucketsPageHandler,
	},
	Route{
		"GET",
		"/buckets/{bucketName}",
		bucketPageHandler,
	},
	Route{
		"POST",
		"/api/buckets",
		createBucketHandler,
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}",
		deleteBucketHandler,
	},
	Route{
		"GET",
		"/api/buckets/{bucketName}/objects/{objectName}",
		getObjectHandler,
	},
	Route{
		"POST",
		"/api/buckets/{bucketName}/objects",
		createObjectHandler,
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}/objects/{objectName}",
		deleteObjectHandler,
	},
}
