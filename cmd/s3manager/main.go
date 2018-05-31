package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mastertinner/adapters/logging"
	"github.com/mastertinner/s3manager/internal/app/s3manager"
	"github.com/matryer/way"
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

	tmplDir := filepath.Join("web", "template")

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
	r := way.NewRouter()
	r.Handle(http.MethodGet, "/", http.RedirectHandler("/buckets", http.StatusPermanentRedirect))
	r.Handle(http.MethodGet, "/buckets", s3manager.HandleBucketsView(s3, tmplDir))
	r.Handle(http.MethodGet, "/buckets/:bucketName", s3manager.HandleBucketView(s3, tmplDir))
	r.Handle(http.MethodPost, "/api/buckets", s3manager.HandleCreateBucket(s3))
	r.Handle(http.MethodDelete, "/api/buckets/:bucketName", s3manager.HandleDeleteBucket(s3))
	r.Handle(http.MethodPost, "/api/buckets/:bucketName/objects", s3manager.HandleCreateObject(s3))
	r.Handle(http.MethodGet, "/api/buckets/:bucketName/objects/:objectName", s3manager.HandleGetObject(s3))
	r.Handle(http.MethodDelete, "/api/buckets/:bucketName/objects/:objectName", s3manager.HandleDeleteObject(s3))

	lr := logging.Handler(os.Stdout)(r)
	log.Fatal(http.ListenAndServe(":"+*port, lr))
}
