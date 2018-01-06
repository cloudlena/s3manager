package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mastertinner/adapters/logging"
	"github.com/mastertinner/s3manager"
	minio "github.com/minio/minio-go"
	"github.com/pkg/errors"
)

func main() {
	var (
		port            = flag.String("port", "8080", "the port the app should listen on")
		endpoint        = flag.String("endpoint", "s3.amazonaws.com", "the s3 endpoint to use")
		accessKeyID     = flag.String("access-key-id", "", "your s3 access key ID")
		secretAccessKey = flag.String("secret-access-key", "", "your s3 secret access key")
		v2Signing       = flag.Bool("v2-signing", false, "set this flag if your S3 provider still uses V2 signing")
	)
	flag.Parse()

	if *accessKeyID == "" || *secretAccessKey == "" {
		flag.Usage()
		os.Exit(2)
	}

	// Set up S3 client
	var s3 *minio.Client
	var err error
	if *v2Signing {
		s3, err = minio.NewV2(*endpoint, *accessKeyID, *secretAccessKey, true)
	} else {
		s3, err = minio.New(*endpoint, *accessKeyID, *secretAccessKey, true)
	}
	if err != nil {
		log.Fatalln(errors.Wrap(err, "error creating s3 client"))
	}

	// Set up router
	r := mux.NewRouter().StrictSlash(true)
	r.
		Methods(http.MethodGet).
		Path("/").
		Handler(http.RedirectHandler("/buckets", http.StatusPermanentRedirect))
	r.
		Methods(http.MethodGet).
		Path("/buckets").
		Handler(s3manager.BucketsViewHandler(s3))
	r.
		Methods(http.MethodGet).
		Path("/buckets/{bucketName}").
		Handler(s3manager.BucketViewHandler(s3))
	r.
		Methods(http.MethodPost).
		Path("/api/buckets").
		Handler(s3manager.CreateBucketHandler(s3))
	r.
		Methods(http.MethodDelete).
		Path("/api/buckets/{bucketName}").
		Handler(s3manager.DeleteBucketHandler(s3))
	r.
		Methods(http.MethodPost).
		Headers(s3manager.HeaderContentType, s3manager.ContentTypeJSON).
		Path("/api/buckets/{bucketName}/objects").
		Handler(s3manager.CopyObjectHandler(s3))
	r.
		Methods(http.MethodPost).
		HeadersRegexp(s3manager.HeaderContentType, s3manager.ContentTypeMultipartForm).
		Path("/api/buckets/{bucketName}/objects").
		Handler(s3manager.CreateObjectHandler(s3))
	r.
		Methods(http.MethodGet).
		Path("/api/buckets/{bucketName}/objects/{objectName}").
		Handler(s3manager.GetObjectHandler(s3))
	r.
		Methods(http.MethodDelete).
		Path("/api/buckets/{bucketName}/objects/{objectName}").
		Handler(s3manager.DeleteObjectHandler(s3))

	log.Fatal(http.ListenAndServe(":"+*port, logging.Handler(os.Stdout)(r)))
}
