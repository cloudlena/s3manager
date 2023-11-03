package main

import (
	"crypto/tls"
	"fmt"
	"github.com/cloudlena/adapters/cors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudlena/adapters/logging"
	"github.com/cloudlena/s3manager/internal/app/s3manager"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

type configuration struct {
	Endpoint            string
	UseIam              bool
	IamEndpoint         string
	AccessKeyID         string
	SecretAccessKey     string
	Region              string
	AllowDelete         bool
	ForceDownload       bool
	UseSSL              bool
	SkipSSLVerification bool
	SignatureType       string
	ListRecursive       bool

	// Addr is the address to listen on
	Addr    string
	Port    string
	Timeout int32
	SseType string
	SseKey  string
}

func parseConfiguration() configuration {
	var accessKeyID, secretAccessKey, iamEndpoint string

	viper.AutomaticEnv()

	viper.SetDefault("ENDPOINT", "s3.amazonaws.com")
	endpoint := viper.GetString("ENDPOINT")

	useIam := viper.GetBool("USE_IAM")

	if useIam {
		iamEndpoint = viper.GetString("IAM_ENDPOINT")
	} else {
		accessKeyID = viper.GetString("ACCESS_KEY_ID")
		if len(accessKeyID) == 0 {
			log.Fatal("please provide ACCESS_KEY_ID")
		}

		secretAccessKey = viper.GetString("SECRET_ACCESS_KEY")
		if len(secretAccessKey) == 0 {
			log.Fatal("please provide SECRET_ACCESS_KEY")
		}
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

	viper.SetDefault("SIGNATURE_TYPE", "V4")
	signatureType := viper.GetString("SIGNATURE_TYPE")

	listRecursive := viper.GetBool("LIST_RECURSIVE")

	viper.SetDefault("PORT", "8080")
	port := viper.GetString("PORT")

	viper.SetDefault("ADDR", "127.0.0.1")
	addr := viper.GetString("ADDR")

	viper.SetDefault("TIMEOUT", 600)
	timeout := viper.GetInt32("TIMEOUT")

	viper.SetDefault("SSE_TYPE", "")
	sseType := viper.GetString("SSE_TYPE")

	viper.SetDefault("SSE_KEY", "")
	sseKey := viper.GetString("SSE_KEY")

	return configuration{
		Endpoint:            endpoint,
		UseIam:              useIam,
		IamEndpoint:         iamEndpoint,
		AccessKeyID:         accessKeyID,
		SecretAccessKey:     secretAccessKey,
		Region:              region,
		AllowDelete:         allowDelete,
		ForceDownload:       forceDownload,
		UseSSL:              useSSL,
		SkipSSLVerification: skipSSLVerification,
		SignatureType:       signatureType,
		ListRecursive:       listRecursive,
		Addr:                addr,
		Port:                port,
		Timeout:             timeout,
		SseType:             sseType,
		SseKey:              sseKey,
	}
}

func main() {
	cfg := parseConfiguration()

	sseType := s3manager.SSEType{Type: cfg.SseType, Key: cfg.SseKey}
	serverTimeout := time.Duration(cfg.Timeout) * time.Second

	// Set up S3 client
	opts := &minio.Options{
		Secure: cfg.UseSSL,
	}
	if cfg.UseIam {
		opts.Creds = credentials.NewIAM(cfg.IamEndpoint)
	} else {
		var signatureType credentials.SignatureType

		switch cfg.SignatureType {
		case "V2":
			signatureType = credentials.SignatureV2
		case "V4":
			signatureType = credentials.SignatureV4
		case "V4Streaming":
			signatureType = credentials.SignatureV4Streaming
		case "Anonymous":
			signatureType = credentials.SignatureAnonymous
		default:
			log.Fatalf("Invalid SIGNATURE_TYPE: %s", cfg.SignatureType)
		}

		opts.Creds = credentials.NewStatic(cfg.AccessKeyID, cfg.SecretAccessKey, "", signatureType)
	}

	if cfg.Region != "" {
		opts.Region = cfg.Region
	}
	if cfg.UseSSL && cfg.SkipSSLVerification {
		opts.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	}
	s3, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		log.Fatalln(fmt.Errorf("error creating s3 client: %w", err))
	}

	// Set up router
	r := mux.NewRouter()
	corsHandler := cors.Handler(cors.Options{
		Methods: []string{"GET", "HEAD", "POST", "PUT", "OPTIONS"},
		Origins: []string{"*"},
	})

	r.Handle("/api/buckets", s3manager.HandleListBuckets(s3)).Methods(http.MethodGet)
	r.Handle("/api/buckets", s3manager.HandleCreateBucket(s3)).Methods(http.MethodPost)
	if cfg.AllowDelete {
		r.Handle("/api/buckets/{bucketName}", s3manager.HandleDeleteBucket(s3)).Methods(http.MethodDelete)
	}
	r.Handle("/api/buckets/{bucketName}/objects", s3manager.HandleCreateObject(s3, sseType)).Methods(http.MethodPost)
	r.Handle("/api/buckets/{bucketName}/list/{objectName:.*}", s3manager.HandleListObjects(s3)).Methods(http.MethodGet)
	r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}/url", s3manager.HandleGenerateUrl(s3)).Methods(http.MethodGet)
	r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}", s3manager.HandleGetObject(s3, cfg.ForceDownload)).Methods(http.MethodGet)
	if cfg.AllowDelete {
		r.Handle("/api/buckets/{bucketName}/objects/{objectName:.*}", s3manager.HandleDeleteObject(s3)).Methods(http.MethodDelete)
	}

	lr := logging.Handler(os.Stdout)(r)

	log.Println("Listening on " + cfg.Addr + ":" + cfg.Port)
	srv := &http.Server{
		Addr:         cfg.Addr + ":" + cfg.Port,
		Handler:      corsHandler(lr),
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
	}
	log.Fatal(srv.ListenAndServe())
}
