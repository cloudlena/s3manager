package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mastertinner/adapters"
	"github.com/mastertinner/adapters/logging"
	. "github.com/mastertinner/s3manager"
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

	// Set up logger
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// Set up router
	r := mux.NewRouter().StrictSlash(true)
	r.
		Methods(http.MethodGet).
		Path("/").
		Handler(adapters.Adapt(
			http.RedirectHandler("/buckets", http.StatusPermanentRedirect),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodGet).
		Path("/buckets").
		Handler(adapters.Adapt(
			BucketsViewHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodGet).
		Path("/buckets/{bucketName}").
		Handler(adapters.Adapt(
			BucketViewHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodPost).
		Path("/api/buckets").
		Handler(adapters.Adapt(
			CreateBucketHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodDelete).
		Path("/api/buckets/{bucketName}").
		Handler(adapters.Adapt(
			DeleteBucketHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodPost).
		Headers(HeaderContentType, ContentTypeJSON).
		Path("/api/buckets/{bucketName}/objects").
		Handler(adapters.Adapt(
			CopyObjectHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodPost).
		HeadersRegexp(HeaderContentType, ContentTypeMultipartForm).
		Path("/api/buckets/{bucketName}/objects").
		Handler(adapters.Adapt(
			CreateObjectHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodGet).
		Path("/api/buckets/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			GetObjectHandler(s3),
			logging.Handler(logger),
		))
	r.
		Methods(http.MethodDelete).
		Path("/api/buckets/{bucketName}/objects/{objectName}").
		Handler(adapters.Adapt(
			DeleteObjectHandler(s3),
			logging.Handler(logger),
		))

	log.Fatal(http.ListenAndServe(":"+*port, r))
}
