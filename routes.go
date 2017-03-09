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
		Chain(IndexPageHandler, Logger()),
	},
	Route{
		"GET",
		"/buckets",
		Chain(s.BucketsPageHandler, Logger()),
	},
	Route{
		"GET",
		"/buckets/{bucketName}",
		Chain(s.BucketPageHandler, Logger()),
	},
	Route{
		"POST",
		"/api/buckets",
		Chain(s.CreateBucketHandler, Logger()),
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}",
		Chain(s.DeleteBucketHandler, Logger()),
	},
	Route{
		"GET",
		"/api/buckets/{bucketName}/objects/{objectName}",
		Chain(s.GetObjectHandler, Logger()),
	},
	Route{
		"POST",
		"/api/buckets/{bucketName}/objects",
		Chain(s.CreateObjectHandler, Logger()),
	},
	Route{
		"DELETE",
		"/api/buckets/{bucketName}/objects/{objectName}",
		Chain(s.DeleteObjectHandler, Logger()),
	},
}
