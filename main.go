package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/cloudlena/adapters/logging"
	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

//go:embed web/template
var templateFS embed.FS

//go:embed web/static
var staticFS embed.FS

type configuration struct {
	S3Instances []s3manager.S3InstanceConfig
	Options     s3manager.Options
	Port        string
	Timeout     time.Duration
}

func parseConfiguration() configuration {
	viper.AutomaticEnv()

	viper.SetDefault("ALLOW_DELETE", true)
	viper.SetDefault("FORCE_DOWNLOAD", true)
	viper.SetDefault("SHOW_METADATA", true)
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("TIMEOUT", 600)

	// A root URL lets the app be served behind a reverse proxy under a path
	// prefix. It is inserted into every link the templates render.
	rootURL := viper.GetString("ROOT_URL")
	if rootURL != "" && !strings.HasPrefix(rootURL, "/") {
		rootURL = "/" + rootURL
	}

	return configuration{
		S3Instances: parseS3Instances(),
		Options: s3manager.Options{
			RootURL:       rootURL,
			BucketName:    viper.GetString("BUCKET_NAME"),
			AllowDelete:   viper.GetBool("ALLOW_DELETE"),
			ForceDownload: viper.GetBool("FORCE_DOWNLOAD"),
			ListRecursive: viper.GetBool("LIST_RECURSIVE"),
			ShowVersions:  viper.GetBool("SHOW_VERSIONS"),
			ShowMetadata:  viper.GetBool("SHOW_METADATA"),
			SSE: s3manager.SSEType{
				Type: viper.GetString("SSE_TYPE"),
				Key:  viper.GetString("SSE_KEY"),
			},
		},
		Port:    viper.GetString("PORT"),
		Timeout: time.Duration(viper.GetInt("TIMEOUT")) * time.Second,
	}
}

// parseS3Instances reads the S3 instances from numbered environment variables
// (S3_1_NAME, S3_1_ENDPOINT, …), stopping at the first number that has no
// NAME. A single instance may also be configured without a number, in which
// case it is named "Default".
func parseS3Instances() []s3manager.S3InstanceConfig {
	var instances []s3manager.S3InstanceConfig

	for i := 1; ; i++ {
		prefix := fmt.Sprintf("S3_%d_", i)
		name := viper.GetString(prefix + "NAME")
		if i == 1 && name == "" {
			// The unnumbered form is how earlier versions of the app were
			// configured.
			prefix, name = "", "Default"
		}
		if name == "" {
			return instances
		}

		viper.SetDefault(prefix+"ENDPOINT", "s3.amazonaws.com")
		viper.SetDefault(prefix+"USE_SSL", true)
		viper.SetDefault(prefix+"SIGNATURE_TYPE", "V4")
		viper.SetDefault(prefix+"BUCKET_LOOKUP", "Auto")

		instance := s3manager.S3InstanceConfig{
			Name:                name,
			Endpoint:            viper.GetString(prefix + "ENDPOINT"),
			UseIam:              viper.GetBool(prefix + "USE_IAM"),
			IamEndpoint:         viper.GetString(prefix + "IAM_ENDPOINT"),
			AccessKeyID:         viper.GetString(prefix + "ACCESS_KEY_ID"),
			SecretAccessKey:     viper.GetString(prefix + "SECRET_ACCESS_KEY"),
			Region:              viper.GetString(prefix + "REGION"),
			UseSSL:              viper.GetBool(prefix + "USE_SSL"),
			SkipSSLVerification: viper.GetBool(prefix + "SKIP_SSL_VERIFICATION"),
			SignatureType:       viper.GetString(prefix + "SIGNATURE_TYPE"),
			BucketLookup:        viper.GetString(prefix + "BUCKET_LOOKUP"),
			PublicURL:           viper.GetString(prefix + "PUBLIC_URL"),
			Buckets:             parseList(viper.GetString(prefix + "BUCKETS")),
		}

		// Anonymous requests are not signed, so they need no key pair. That is
		// how public buckets are browsed.
		if !instance.UseIam && instance.SignatureType != "Anonymous" {
			if instance.AccessKeyID == "" {
				log.Fatalf("please provide %sACCESS_KEY_ID for instance %s", prefix, name)
			}
			if instance.SecretAccessKey == "" {
				log.Fatalf("please provide %sSECRET_ACCESS_KEY for instance %s", prefix, name)
			}
		}

		instances = append(instances, instance)
	}
}

// parseList splits a comma-separated configuration value, dropping blanks.
func parseList(value string) []string {
	var items []string
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}

	return items
}

func main() {
	configuration := parseConfiguration()
	opts := configuration.Options

	templates, err := fs.Sub(templateFS, "web/template")
	if err != nil {
		log.Fatal(err)
	}
	statics, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		log.Fatal(err)
	}

	instances, err := s3manager.NewS3Instances(configuration.S3Instances)
	if err != nil {
		log.Fatal(fmt.Errorf("error creating S3 instances: %w", err))
	}

	// withInstance binds a handler that operates on a single S3 client to the
	// instance addressed by the request.
	withInstance := func(handler func(s3manager.S3) http.HandlerFunc) http.Handler {
		return s3manager.WithInstance(instances, handler)
	}

	r := mux.NewRouter()

	// The root redirects to the first instance's bucket list.
	r.Handle("/", http.RedirectHandler(opts.RootURL+"/"+instances[0].Name+"/buckets", http.StatusPermanentRedirect)).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(statics)))).Methods(http.MethodGet)
	r.Handle("/api/s3-instances", s3manager.HandleGetS3Instances(instances)).Methods(http.MethodGet)

	// S3 management endpoints, all scoped to an instance.
	r.Handle("/{instance}/buckets", s3manager.HandleBucketsView(instances, templates, opts)).Methods(http.MethodGet)
	r.PathPrefix("/{instance}/buckets/").Handler(s3manager.HandleBucketView(instances, templates, opts)).Methods(http.MethodGet)
	r.Handle("/{instance}/api/buckets", withInstance(s3manager.HandleCreateBucket)).Methods(http.MethodPost)
	r.Handle("/{instance}/api/buckets/{bucketName}/objects", withInstance(func(s3 s3manager.S3) http.HandlerFunc {
		return s3manager.HandleCreateObject(s3, opts.SSE)
	})).Methods(http.MethodPost)
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/bulk-download", withInstance(s3manager.HandleBulkDownloadObjects)).Methods(http.MethodPost)
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/url", withInstance(s3manager.HandleGenerateURL)).Methods(http.MethodGet)
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/public-access", s3manager.HandleCheckPublicAccess(instances)).Methods(http.MethodGet)
	if opts.ShowMetadata {
		r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/metadata", withInstance(s3manager.HandleGetObjectMetadata)).Methods(http.MethodGet)
	}
	r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}", withInstance(func(s3 s3manager.S3) http.HandlerFunc {
		return s3manager.HandleGetObject(s3, opts)
	})).Methods(http.MethodGet)
	if opts.AllowDelete {
		r.Handle("/{instance}/api/buckets/{bucketName}", withInstance(s3manager.HandleDeleteBucket)).Methods(http.MethodDelete)
		r.Handle("/{instance}/api/buckets/{bucketName}/objects/bulk-delete", withInstance(s3manager.HandleBulkDeleteObjects)).Methods(http.MethodPost)
		r.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}", withInstance(s3manager.HandleDeleteObject)).Methods(http.MethodDelete)
	}
	r.Handle("/{instance}/api/buckets/{bucketName}/policy", withInstance(s3manager.HandleGetBucketPolicy)).Methods(http.MethodGet)
	r.Handle("/{instance}/api/buckets/{bucketName}/policy", withInstance(s3manager.HandlePutBucketPolicy)).Methods(http.MethodPut)

	srv := &http.Server{
		Addr:         ":" + configuration.Port,
		Handler:      logging.Handler(os.Stdout)(r),
		ReadTimeout:  configuration.Timeout,
		WriteTimeout: configuration.Timeout,
	}

	log.Printf("serving %d S3 instance(s) on port %s", len(instances), configuration.Port)
	log.Fatal(srv.ListenAndServe())
}
