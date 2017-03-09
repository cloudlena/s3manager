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

var s = &Server{
	s3: NewMinioClient(),
}

var routes = Routes{
	Route{
		"GET",
		"/",
		Chain(indexPageHandler, Logger()),
	},
	Route{
		"GET",
		"/buckets",
		Chain(s.bucketsPageHandler, Logger()),
	},
	Route{
		"GET",
		"/buckets/{bucketName}",
		Chain(s.bucketPageHandler, Logger()),
	},
	Route{
		"POST",
		"/api/buckets",
		Chain(s.CreateBucketHandler, Logger()),
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}",
		Chain(s.deleteBucketHandler, Logger()),
	},
	Route{
		"GET",
		"/api/buckets/{bucketName}/objects/{objectName}",
		Chain(s.getObjectHandler, Logger()),
	},
	Route{
		"POST",
		"/api/buckets/{bucketName}/objects",
		Chain(s.createObjectHandler, Logger()),
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}/objects/{objectName}",
		Chain(s.deleteObjectHandler, Logger()),
	},
}
