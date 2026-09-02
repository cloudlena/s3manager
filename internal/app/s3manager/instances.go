package s3manager

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3InstanceConfig holds the configuration of a single S3 instance.
type S3InstanceConfig struct {
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

// S3Instance is a configured S3 instance and its client.
type S3Instance struct {
	ID     string
	Name   string
	Client S3
}

// S3Instances is the ordered set of configured S3 instances. It is built once
// at start-up and never modified, so handlers can read it concurrently.
type S3Instances []*S3Instance

// signatureTypes maps the SIGNATURE_TYPE configuration value to its S3 equivalent.
var signatureTypes = map[string]credentials.SignatureType{
	"V2":          credentials.SignatureV2,
	"V4":          credentials.SignatureV4,
	"V4Streaming": credentials.SignatureV4Streaming,
	"Anonymous":   credentials.SignatureAnonymous,
}

// NewS3Instances creates a client for every given configuration. Instances are
// addressable by their name and by their position in the list, starting at 1.
func NewS3Instances(configs []S3InstanceConfig) (S3Instances, error) {
	if len(configs) == 0 {
		return nil, errors.New("no S3 instances configured")
	}

	instances := make(S3Instances, 0, len(configs))
	for i, config := range configs {
		client, err := newS3Client(config)
		if err != nil {
			return nil, err
		}
		instances = append(instances, &S3Instance{
			ID:     strconv.Itoa(i + 1),
			Name:   config.Name,
			Client: client,
		})
	}

	return instances, nil
}

// newS3Client creates the S3 client described by a single instance configuration.
func newS3Client(config S3InstanceConfig) (S3, error) {
	opts := &minio.Options{
		Secure: config.UseSSL,
		Region: config.Region,
	}

	if config.UseIam {
		opts.Creds = credentials.NewIAM(config.IamEndpoint)
	} else {
		signatureType, ok := signatureTypes[config.SignatureType]
		if !ok {
			return nil, fmt.Errorf("invalid SIGNATURE_TYPE: %s", config.SignatureType)
		}
		opts.Creds = credentials.NewStatic(config.AccessKeyID, config.SecretAccessKey, "", signatureType)
	}

	if config.UseSSL && config.SkipSSLVerification {
		opts.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
	}

	client, err := minio.New(config.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating s3 client for instance %s: %w", config.Name, err)
	}

	return client, nil
}

// Get returns the instance with the given ID or name.
func (instances S3Instances) Get(identifier string) (*S3Instance, error) {
	for _, instance := range instances {
		if instance.ID == identifier {
			return instance, nil
		}
	}
	for _, instance := range instances {
		if instance.Name == identifier {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("S3 instance '%s' not found", identifier)
}

// WithInstance resolves the instance addressed by the request's {instance}
// variable and delegates to a handler for that instance's client.
func WithInstance(instances S3Instances, handler func(S3) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instance, ok := resolveInstance(w, r, instances)
		if !ok {
			return
		}

		handler(instance.Client)(w, r)
	}
}

// resolveInstance looks up the instance addressed by the request's {instance}
// variable, responding with a not found error if there is no such instance.
func resolveInstance(w http.ResponseWriter, r *http.Request, instances S3Instances) (*S3Instance, bool) {
	instance, err := instances.Get(mux.Vars(r)["instance"])
	if err != nil {
		http.Error(w, fmt.Sprintf("Instance not found: %s", err), http.StatusNotFound)
		return nil, false
	}

	return instance, true
}

// HandleGetS3Instances lists all configured S3 instances.
func HandleGetS3Instances(instances S3Instances) http.HandlerFunc {
	type instanceInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		infos := make([]instanceInfo, len(instances))
		for i, instance := range instances {
			infos[i] = instanceInfo{ID: instance.ID, Name: instance.Name}
		}

		writeJSON(w, http.StatusOK, struct {
			Instances []instanceInfo `json:"instances"`
		}{Instances: infos})
	}
}
