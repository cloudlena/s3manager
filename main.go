package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudlena/adapters/logging"
	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

//go:embed web/template
var templateFS embed.FS

//go:embed web/static
var staticFS embed.FS

type s3InstanceConfig struct {
	Name                string
	Endpoint            string
	UseIam              bool
	IamEndpoint         string
	AccessKeyID         string
	SecretAccessKey     string
	Region              string
	UseSSL              bool
	SkipSSLVerification bool
	SignatureType       string
}

type configuration struct {
	S3Instances     []s3InstanceConfig
	AllowDelete     bool
	ForceDownload   bool
	ListRecursive   bool
	Port            string
	Timeout         int32
	SseType         string
	SseKey          string
	Username        string
	Password        string
}

func parseConfiguration() configuration {
	viper.AutomaticEnv()

	// Parse S3 instances from numbered environment variables
	var s3Instances []s3InstanceConfig
	for i := 1; ; i++ {
		prefix := fmt.Sprintf("%d_", i)
		name := viper.GetString(prefix + "NAME")
		endpoint := viper.GetString(prefix + "ENDPOINT")
		
		// If NAME or ENDPOINT is not found, stop parsing
		if name == "" || endpoint == "" {
			break
		}

		accessKeyID := viper.GetString(prefix + "ACCESS_KEY_ID")
		secretAccessKey := viper.GetString(prefix + "SECRET_ACCESS_KEY")
		useIam := viper.GetBool(prefix + "USE_IAM")
		iamEndpoint := viper.GetString(prefix + "IAM_ENDPOINT")
		region := viper.GetString(prefix + "REGION")
		
		viper.SetDefault(prefix+"USE_SSL", true)
		useSSL := viper.GetBool(prefix + "USE_SSL")
		
		viper.SetDefault(prefix+"SKIP_SSL_VERIFICATION", false)
		skipSSLVerification := viper.GetBool(prefix + "SKIP_SSL_VERIFICATION")
		
		viper.SetDefault(prefix+"SIGNATURE_TYPE", "V4")
		signatureType := viper.GetString(prefix + "SIGNATURE_TYPE")

		if !useIam {
			if accessKeyID == "" {
				log.Fatalf("please provide %sACCESS_KEY_ID for instance %s", prefix, name)
			}
			if secretAccessKey == "" {
				log.Fatalf("please provide %sSECRET_ACCESS_KEY for instance %s", prefix, name)
			}
		}

		s3Instances = append(s3Instances, s3InstanceConfig{
			Name:                name,
			Endpoint:            endpoint,
			UseIam:              useIam,
			IamEndpoint:         iamEndpoint,
			AccessKeyID:         accessKeyID,
			SecretAccessKey:     secretAccessKey,
			Region:              region,
			UseSSL:              useSSL,
			SkipSSLVerification: skipSSLVerification,
			SignatureType:       signatureType,
		})
	}

	if len(s3Instances) == 0 {
		log.Fatal("no S3 instances configured. Please provide numbered environment variables like 1_NAME, 1_ENDPOINT, etc.")
	}

	viper.SetDefault("ALLOW_DELETE", true)
	allowDelete := viper.GetBool("ALLOW_DELETE")

	viper.SetDefault("FORCE_DOWNLOAD", true)
	forceDownload := viper.GetBool("FORCE_DOWNLOAD")

	listRecursive := viper.GetBool("LIST_RECURSIVE")

	viper.SetDefault("PORT", "8080")
	port := viper.GetString("PORT")

	viper.SetDefault("TIMEOUT", 600)
	timeout := viper.GetInt32("TIMEOUT")

	viper.SetDefault("SSE_TYPE", "")
	sseType := viper.GetString("SSE_TYPE")

	viper.SetDefault("SSE_KEY", "")
	sseKey := viper.GetString("SSE_KEY")

	// Parse optional authentication credentials (base64 encoded)
	var username, password string
	usernameB64 := viper.GetString("USERNAME")
	passwordB64 := viper.GetString("PASSWORD")
	if usernameB64 != "" && passwordB64 != "" {
		usernameBytes, err := base64.StdEncoding.DecodeString(usernameB64)
		if err != nil {
			log.Fatalf("USERNAME is not valid base64: %v", err)
		}
		passwordBytes, err := base64.StdEncoding.DecodeString(passwordB64)
		if err != nil {
			log.Fatalf("PASSWORD is not valid base64: %v", err)
		}
		username = string(usernameBytes)
		password = string(passwordBytes)
	}

	return configuration{
		S3Instances:   s3Instances,
		AllowDelete:   allowDelete,
		ForceDownload: forceDownload,
		ListRecursive: listRecursive,
		Port:          port,
		Timeout:       timeout,
		SseType:       sseType,
		SseKey:        sseKey,
		Username:      username,
		Password:      password,
	}
}

func main() {
	configuration := parseConfiguration()

	sseType := s3manager.SSEType{Type: configuration.SseType, Key: configuration.SseKey}
	serverTimeout := time.Duration(configuration.Timeout) * time.Second

	// Set up templates
	templates, err := fs.Sub(templateFS, "web/template")
	if err != nil {
		log.Fatal(err)
	}
	// Set up statics
	statics, err := fs.Sub(staticFS, "web/static")
	if err != nil {
		log.Fatal(err)
	}

	// Convert configuration to s3manager format
	var s3Configs []s3manager.S3InstanceConfig
	for _, instance := range configuration.S3Instances {
		s3Configs = append(s3Configs, s3manager.S3InstanceConfig{
			Name:                instance.Name,
			Endpoint:            instance.Endpoint,
			UseIam:              instance.UseIam,
			IamEndpoint:         instance.IamEndpoint,
			AccessKeyID:         instance.AccessKeyID,
			SecretAccessKey:     instance.SecretAccessKey,
			Region:              instance.Region,
			UseSSL:              instance.UseSSL,
			SkipSSLVerification: instance.SkipSSLVerification,
			SignatureType:       instance.SignatureType,
		})
	}

	// Set up Multi S3 Manager
	s3Manager, err := s3manager.NewMultiS3Manager(s3Configs)
	if err != nil {
		log.Fatalln(fmt.Errorf("error creating multi s3 manager: %w", err))
	}

	// Check for a root URL to insert into HTML templates in case of reverse proxying
	rootURL, rootSet := os.LookupEnv("ROOT_URL")
	if rootSet && !strings.HasPrefix(rootURL, "/") {
		rootURL = "/" + rootURL
	}

	// Set up router
	r := mux.NewRouter()

	// Generate a random session secret for HMAC-signed cookies
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		log.Fatalf("failed to generate session secret: %v", err)
	}
	sessionSecret := hex.EncodeToString(secretBytes)

	// Authentication routes and middleware (opt-in: only if USERNAME and PASSWORD are set)
	authEnabled := configuration.Username != "" && configuration.Password != ""
	if authEnabled {
		r.Handle("/login", s3manager.HandleLoginPage(templates, rootURL)).Methods(http.MethodGet)
		r.Handle("/api/login", s3manager.HandleLoginSubmit(configuration.Username, configuration.Password, sessionSecret, rootURL)).Methods(http.MethodPost)
		r.Handle("/logout", s3manager.HandleLogout(rootURL)).Methods(http.MethodGet)
		r.Use(s3manager.AuthMiddleware(configuration.Username, configuration.Password, sessionSecret, rootURL))
	}

	r.Handle("/", http.RedirectHandler("/buckets", http.StatusPermanentRedirect)).Methods(http.MethodGet)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(statics)))).Methods(http.MethodGet)
	
	// S3 instance management endpoints
	r.Handle("/api/s3-instances", s3manager.HandleGetS3Instances(s3Manager)).Methods(http.MethodGet)
	r.Handle("/api/s3-instances/{instanceId}/switch", s3manager.HandleSwitchS3Instance(s3Manager)).Methods(http.MethodPost)
	
	// S3 management endpoints (using current instance)
	r.Handle("/buckets", s3manager.HandleBucketsViewWithManager(s3Manager, templates, configuration.AllowDelete, rootURL, authEnabled)).Methods(http.MethodGet)
	r.PathPrefix("/buckets/").Handler(s3manager.HandleBucketViewWithManager(s3Manager, templates, configuration.AllowDelete, configuration.ListRecursive, rootURL, authEnabled)).Methods(http.MethodGet)
	r.Handle("/api/buckets", s3manager.HandleCreateBucketWithManager(s3Manager)).Methods(http.MethodPost)
	if configuration.AllowDelete {
		r.Handle("/api/buckets/{bucketName}", s3manager.HandleDeleteBucketWithManager(s3Manager)).Methods(http.MethodDelete)
	}
	r.Handle("/api/buckets/{bucketName}/objects", s3manager.HandleCreateObjectWithManager(s3Manager, sseType)).Methods(http.MethodPost)
	r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}/move", s3manager.HandleMoveObjectWithManager(s3Manager, sseType)).Methods(http.MethodPost)
	r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}/url", s3manager.HandleGenerateURLWithManager(s3Manager)).Methods(http.MethodGet)
	r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}", s3manager.HandleGetObjectWithManager(s3Manager, configuration.ForceDownload)).Methods(http.MethodGet)
	if configuration.AllowDelete {
		r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}", s3manager.HandleDeleteObjectWithManager(s3Manager)).Methods(http.MethodDelete)
	}

	lr := logging.Handler(os.Stdout)(r)
	srv := &http.Server{
		Addr:         ":" + configuration.Port,
		Handler:      lr,
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}
