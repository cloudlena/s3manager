package main

import (
	"fmt"
	"html/template"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	minio "github.com/minio/minio-go"
)

// ObjectWithIcon is a minio object with an added icon
type ObjectWithIcon struct {
	minio.ObjectInfo
	Icon string
}

// BucketPage defines the details page of a bucket
type BucketPage struct {
	BucketName string
	Objects    []ObjectWithIcon
}

// indexPageHandler forwards to "/buckets"
func indexPageHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/buckets", http.StatusPermanentRedirect)
}

// bucketsHandler handles the main page
func bucketsPageHandler(w http.ResponseWriter, r *http.Request) {
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

// bucketHandler shows the details page of a bucket
func bucketPageHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucketName"]
	var objects []ObjectWithIcon

	lp := path.Join("templates", "layout.html")
	bp := path.Join("templates", "bucket.html")

	t, err := template.ParseFiles(lp, bp)
	if err != nil {
		panic(err)
	}

	doneCh := make(chan struct{})

	objectCh := minioClient.ListObjectsV2(bucketName, "", false, doneCh)
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		objectWithIcon := ObjectWithIcon{object, getIcon(object.Key)}
		objects = append(objects, objectWithIcon)
	}

	bucketPage := BucketPage{
		BucketName: bucketName,
		Objects:    objects,
	}

	err = t.ExecuteTemplate(w, "layout", bucketPage)
	if err != nil {
		panic(err)
	}
}

func getIcon(fileName string) string {
	e := path.Ext(fileName)

	switch e {
	case ".tgz":
		return "archive"
	case ".png", ".jpg", ".gif", ".svg":
		return "photo"
	case ".mp3":
		return "music_note"
	}

	return "insert_drive_file"
}
