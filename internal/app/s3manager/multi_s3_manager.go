package s3manager

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Instance represents a configured S3 instance
type S3Instance struct {
	ID     string
	Name   string
	Client S3
}

// MultiS3Manager manages multiple S3 instances
type MultiS3Manager struct {
	instances     map[string]*S3Instance
	instanceOrder []string
	mu            sync.RWMutex
}

// S3InstanceConfig holds configuration for a single S3 instance
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

// NewMultiS3Manager creates a new MultiS3Manager with the given configurations
func NewMultiS3Manager(configs []S3InstanceConfig) (*MultiS3Manager, error) {
	manager := &MultiS3Manager{
		instances:     make(map[string]*S3Instance),
		instanceOrder: make([]string, 0, len(configs)),
	}

	for i, config := range configs {
		instanceID := fmt.Sprintf("%d", i+1)

		// Set up S3 client options
		opts := &minio.Options{
			Secure: config.UseSSL,
		}

		if config.UseIam {
			opts.Creds = credentials.NewIAM(config.IamEndpoint)
		} else {
			var signatureType credentials.SignatureType

			switch config.SignatureType {
			case "V2":
				signatureType = credentials.SignatureV2
			case "V4":
				signatureType = credentials.SignatureV4
			case "V4Streaming":
				signatureType = credentials.SignatureV4Streaming
			case "Anonymous":
				signatureType = credentials.SignatureAnonymous
			default:
				return nil, fmt.Errorf("invalid SIGNATURE_TYPE: %s", config.SignatureType)
			}

			opts.Creds = credentials.NewStatic(config.AccessKeyID, config.SecretAccessKey, "", signatureType)
		}

		if config.Region != "" {
			opts.Region = config.Region
		}

		if config.UseSSL && config.SkipSSLVerification {
			opts.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec
		}

		// Create S3 client
		s3Client, err := minio.New(config.Endpoint, opts)
		if err != nil {
			return nil, fmt.Errorf("error creating s3 client for instance %s: %w", config.Name, err)
		}

		instance := &S3Instance{
			ID:     instanceID,
			Name:   config.Name,
			Client: s3Client,
		}

		manager.instances[instanceID] = instance
		manager.instanceOrder = append(manager.instanceOrder, instanceID)
	}

	if len(manager.instances) == 0 {
		return nil, fmt.Errorf("no S3 instances configured")
	}

	log.Printf("Initialized MultiS3Manager with %d instances", len(manager.instances))
	return manager, nil
}

// GetInstance returns an S3 instance by its ID or Name
func (m *MultiS3Manager) GetInstance(identifier string) (*S3Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try to find by ID first
	if instance, exists := m.instances[identifier]; exists {
		return instance, nil
	}

	// Try to find by Name
	for _, instance := range m.instances {
		if instance.Name == identifier {
			return instance, nil
		}
	}

	return nil, fmt.Errorf("S3 instance '%s' not found", identifier)
}

// GetCurrentClient returns the currently active S3 client
// Deprecated: Use GetInstance instead
func (m *MultiS3Manager) GetCurrentClient() S3 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return first instance as fallback for backwards compatibility
	if len(m.instanceOrder) > 0 {
		return m.instances[m.instanceOrder[0]].Client
	}
	return nil
}

// GetCurrentInstance returns the currently active S3 instance info
// Deprecated: Use GetInstance instead
func (m *MultiS3Manager) GetCurrentInstance() *S3Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return first instance as fallback for backwards compatibility
	if len(m.instanceOrder) > 0 {
		return m.instances[m.instanceOrder[0]]
	}
	return nil
}

// SetCurrentInstance switches to the specified S3 instance
// Deprecated: Instance selection is now URL-based
func (m *MultiS3Manager) SetCurrentInstance(instanceID string) error {
	return fmt.Errorf("instance switching is no longer supported - use URL-based instance selection")
}

// GetAllInstances returns all available S3 instances
func (m *MultiS3Manager) GetAllInstances() []*S3Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instances := make([]*S3Instance, 0, len(m.instanceOrder))
	for _, id := range m.instanceOrder {
		instances = append(instances, m.instances[id])
	}
	return instances
}
