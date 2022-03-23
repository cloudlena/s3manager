package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/cloudlena/adapters/logging"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"

	"github.com/cloudlena/s3manager/internal/app/s3manager"
)

//go:embed web/template
var templateFS embed.FS

func main() {
	viper.AutomaticEnv()

	viper.SetDefault("ENDPOINT", "s3.amazonaws.com")
	endpoint := viper.GetString("ENDPOINT")

	accessKeyID := viper.GetString("ACCESS_KEY_ID")
	if len(accessKeyID) == 0 {
		log.Fatal("please provide ACCESS_KEY_ID")
	}

	secretAccessKey := viper.GetString("SECRET_ACCESS_KEY")
	if len(secretAccessKey) == 0 {
		log.Fatal("please provide SECRET_ACCESS_KEY")
	}

	region := viper.GetString("REGION")

	viper.SetDefault("ALLOW_DELETE", true)
	allowDelete := viper.GetBool("ALLOW_DELETE")

	viper.SetDefault("FORCE_DOWNLOAD", true)
	forceDownload := viper.GetBool("FORCE_DOWNLOAD")

	viper.SetDefault("USE_SSL", true)
	useSSL := viper.GetBool("USE_SSL")

	viper.SetDefault("SKIP_SSL_VERIFICATION", false)
	skipSSLVerification := viper.GetBool("SKIP_SSL_VERIFICATION")

	listRecursive := viper.GetBool("LIST_RECURSIVE")

	viper.SetDefault("PORT", "8080")
	port := viper.GetString("PORT")

	// Set up templates
	templates, err := fs.Sub(templateFS, "web/template")
	if err != nil {
		log.Fatal(err)
	}

	// Set up S3 client
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	}
	if region != "" {
		opts.Region = region
	}
	if useSSL && skipSSLVerification {
		opts.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	}
	s3, err := minio.New(endpoint, opts)
	if err != nil {
		log.Fatalln(fmt.Errorf("error creating s3 client: %w", err))
	}

	// Set up router
	r := mux.NewRouter()
	r.Handle("/", http.RedirectHandler("/buckets", http.StatusPermanentRedirect)).Methods(http.MethodGet)
	r.Handle("/buckets", s3manager.HandleBucketsView(s3, templates, allowDelete)).Methods(http.MethodGet)
	r.Handle("/buckets/{bucketName}", s3manager.HandleBucketView(s3, templates, allowDelete, listRecursive)).Methods(http.MethodGet)
	r.Handle("/api/buckets", s3manager.HandleCreateBucket(s3)).Methods(http.MethodPost)
	if allowDelete {
		r.Handle("/api/buckets/{bucketName}", s3manager.HandleDeleteBucket(s3)).Methods(http.MethodDelete)
	}
	r.Handle("/api/buckets/{bucketName}/objects", s3manager.HandleCreateObject(s3)).Methods(http.MethodPost)
	r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}", s3manager.HandleGetObject(s3, forceDownload)).Methods(http.MethodGet)
	if allowDelete {
		r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}", s3manager.HandleDeleteObject(s3)).Methods(http.MethodDelete)
	}

	lr := logging.Handler(os.Stdout)(r)
	log.Fatal(http.ListenAndServe(":"+port, lr))
}
