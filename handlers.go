package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path"

	"strings"

	minio "github.com/minio/minio-go"
)

// indexHandler forwards to "/buckets"
func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/buckets", http.StatusPermanentRedirect)
}

// bucketsHandler handles the main page
func bucketsHandler(w http.ResponseWriter, r *http.Request) {
	lp := path.Join("templates", "layout.html")
	ip := path.Join("templates", "index.html")

	t, err := template.ParseFiles(lp, ip)
	if err != nil {
		panic(err)
	}

	buckets, err := minioClient.ListBuckets()
	if err != nil {
		panic(err)
	}

	err = t.ExecuteTemplate(w, "layout", buckets)
	if err != nil {
		panic(err)
	}
}

// bucketHandler handles the main page
func bucketHandler(w http.ResponseWriter, r *http.Request) {
	bucket := strings.Split(r.URL.Path, "/")[2]
	var objects []minio.ObjectInfo

	lp := path.Join("templates", "layout.html")
	bp := path.Join("templates", "bucket.html")

	t, err := template.ParseFiles(lp, bp)
	if err != nil {
		panic(err)
	}

	// Create a done channel to control 'ListObjectsV2' go routine.
	doneCh := make(chan struct{})

	objectCh := minioClient.ListObjectsV2(bucket, "", false, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		objects = append(objects, object)
	}

	err = t.ExecuteTemplate(w, "layout", objects)
	if err != nil {
		panic(err)
	}
}
