package main

import (
	"context"
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
	"github.com/cloudlena/s3manager/internal/app/s3manager/auth"
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
	Auth        auth.Config
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
	viper.SetDefault("AUTH_PROVIDER", auth.ProviderNone)
	viper.SetDefault("AUTH_ANONYMOUS_ROLE", "viewer")
	viper.SetDefault("SESSION_MAX_AGE", 28800)
	viper.SetDefault("SESSION_COOKIE_SECURE", true)
	viper.SetDefault("OIDC_SCOPES", "profile,email")
	viper.SetDefault("OIDC_ROLE_CLAIM", "groups")
	viper.SetDefault("OIDC_DEFAULT_ROLE", "none")

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
		Auth:    parseAuthConfiguration(rootURL),
		Port:    viper.GetString("PORT"),
		Timeout: time.Duration(viper.GetInt("TIMEOUT")) * time.Second,
	}
}

// parseAuthConfiguration reads how access to the app itself is guarded. None of
// it touches the S3 credentials, which stay server side.
func parseAuthConfiguration(rootURL string) auth.Config {
	anonymousRole, err := auth.ParseRole(viper.GetString("AUTH_ANONYMOUS_ROLE"))
	if err != nil {
		log.Fatalf("invalid AUTH_ANONYMOUS_ROLE: %s", err)
	}
	defaultRole, err := auth.ParseRole(viper.GetString("OIDC_DEFAULT_ROLE"))
	if err != nil {
		log.Fatalf("invalid OIDC_DEFAULT_ROLE: %s", err)
	}

	return auth.Config{
		Provider:      viper.GetString("AUTH_PROVIDER"),
		AnonymousRole: anonymousRole,
		SessionSecret: viper.GetString("SESSION_SECRET"),
		OIDC: auth.OIDCConfig{
			Issuer:        viper.GetString("OIDC_ISSUER"),
			ClientID:      viper.GetString("OIDC_CLIENT_ID"),
			ClientSecret:  viper.GetString("OIDC_CLIENT_SECRET"),
			RedirectURL:   viper.GetString("OIDC_REDIRECT_URL"),
			Scopes:        splitList(viper.GetString("OIDC_SCOPES")),
			RoleClaim:     viper.GetString("OIDC_ROLE_CLAIM"),
			ViewerGroups:  splitList(viper.GetString("OIDC_VIEWER_GROUPS")),
			WriterGroups:  splitList(viper.GetString("OIDC_WRITER_GROUPS")),
			DefaultRole:   defaultRole,
			SessionMaxAge: viper.GetInt("SESSION_MAX_AGE"),
			RootURL:       rootURL,
			CookieSecure:  viper.GetBool("SESSION_COOKIE_SECURE"),
		},
	}
}

// splitList reads a comma separated configuration value, ignoring empty and
// padded entries.
func splitList(value string) []string {
	var values []string
	for _, entry := range strings.Split(value, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			values = append(values, entry)
		}
	}

	return values
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
		}

		if !instance.UseIam {
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

	authenticator, err := auth.New(context.Background(), configuration.Auth)
	if err != nil {
		log.Fatal(fmt.Errorf("error setting up authentication: %w", err))
	}

	// withInstance binds a handler that operates on a single S3 client to the
	// instance addressed by the request.
	withInstance := func(handler func(s3manager.S3) http.HandlerFunc) http.Handler {
		return s3manager.WithInstance(instances, handler)
	}

	// writer guards the endpoints that change something. It narrows access
	// within what the configuration already allows: an endpoint the
	// configuration disables is not registered at all, so no role can reach it.
	writer := auth.RequireRole(auth.RoleWriter)

	r := mux.NewRouter()

	// Static assets and the login endpoints have to stay reachable without a
	// session, or a user could never log in.
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(statics)))).Methods(http.MethodGet)
	authenticator.RegisterRoutes(r)

	// Everything below is authenticated, and every request carries a role.
	app := r.NewRoute().Subrouter()
	app.Use(auth.Middleware(authenticator))

	// The root redirects to the first instance's bucket list.
	app.Handle("/", http.RedirectHandler(opts.RootURL+"/"+instances[0].Name+"/buckets", http.StatusPermanentRedirect)).Methods(http.MethodGet)
	app.Handle("/api/s3-instances", s3manager.HandleGetS3Instances(instances)).Methods(http.MethodGet)

	// S3 management endpoints, all scoped to an instance.
	app.Handle("/{instance}/buckets", s3manager.HandleBucketsView(instances, templates, opts)).Methods(http.MethodGet)
	app.PathPrefix("/{instance}/buckets/").Handler(s3manager.HandleBucketView(instances, templates, opts)).Methods(http.MethodGet)
	app.Handle("/{instance}/api/buckets", writer(withInstance(s3manager.HandleCreateBucket))).Methods(http.MethodPost)
	app.Handle("/{instance}/api/buckets/{bucketName}/objects", writer(withInstance(func(s3 s3manager.S3) http.HandlerFunc {
		return s3manager.HandleCreateObject(s3, opts.SSE)
	}))).Methods(http.MethodPost)
	app.Handle("/{instance}/api/buckets/{bucketName}/objects/bulk-download", withInstance(s3manager.HandleBulkDownloadObjects)).Methods(http.MethodPost)
	app.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/url", withInstance(s3manager.HandleGenerateURL)).Methods(http.MethodGet)
	app.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/public-access", withInstance(s3manager.HandleCheckPublicAccess)).Methods(http.MethodGet)
	if opts.ShowMetadata {
		app.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}/metadata", withInstance(s3manager.HandleGetObjectMetadata)).Methods(http.MethodGet)
	}
	app.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}", withInstance(func(s3 s3manager.S3) http.HandlerFunc {
		return s3manager.HandleGetObject(s3, opts)
	})).Methods(http.MethodGet)
	if opts.AllowDelete {
		app.Handle("/{instance}/api/buckets/{bucketName}", writer(withInstance(s3manager.HandleDeleteBucket))).Methods(http.MethodDelete)
		app.Handle("/{instance}/api/buckets/{bucketName}/objects/bulk-delete", writer(withInstance(s3manager.HandleBulkDeleteObjects))).Methods(http.MethodPost)
		app.Handle("/{instance}/api/buckets/{bucketName}/objects/{objectName:.*}", writer(withInstance(s3manager.HandleDeleteObject))).Methods(http.MethodDelete)
	}
	app.Handle("/{instance}/api/buckets/{bucketName}/policy", withInstance(s3manager.HandleGetBucketPolicy)).Methods(http.MethodGet)
	app.Handle("/{instance}/api/buckets/{bucketName}/policy", writer(withInstance(s3manager.HandlePutBucketPolicy))).Methods(http.MethodPut)

	srv := &http.Server{
		Addr:         ":" + configuration.Port,
		Handler:      logging.Handler(os.Stdout)(r),
		ReadTimeout:  configuration.Timeout,
		WriteTimeout: configuration.Timeout,
	}

	log.Printf("serving %d S3 instance(s) on port %s", len(instances), configuration.Port)
	log.Fatal(srv.ListenAndServe())
}
