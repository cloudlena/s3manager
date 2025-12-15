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
	currentID     string
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
	HumanReadableSize   bool
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
			ID:                instanceID,
			Name:              config.Name,
			Client:            s3Client,
			HumanReadableSize: config.HumanReadableSize,
		}

		manager.instances[instanceID] = instance
		manager.instanceOrder = append(manager.instanceOrder, instanceID)

		// Set the first instance as current
		if i == 0 {
			manager.currentID = instanceID
		}
	}

	if len(manager.instances) == 0 {
		return nil, fmt.Errorf("no S3 instances configured")
	}

	log.Printf("Initialized MultiS3Manager with %d instances", len(manager.instances))
	return manager, nil
}

// GetCurrentClient returns the currently active S3 client
func (m *MultiS3Manager) GetCurrentClient() S3 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.instances[m.currentID]
	if !exists && len(m.instances) > 0 {
		// Fallback to first instance if current doesn't exist
		m.currentID = m.instanceOrder[0]
		instance = m.instances[m.currentID]
	}
	return instance.Client
}

// GetCurrentInstance returns the currently active S3 instance info
func (m *MultiS3Manager) GetCurrentInstance() *S3Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.instances[m.currentID]
	if !exists && len(m.instances) > 0 {
		// Fallback to first instance if current doesn't exist
		m.currentID = m.instanceOrder[0]
		instance = m.instances[m.currentID]
	}
	return instance
}

// SetCurrentInstance switches to the specified S3 instance
func (m *MultiS3Manager) SetCurrentInstance(instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.instances[instanceID]; !exists {
		return fmt.Errorf("S3 instance with ID %s not found", instanceID)
	}

	m.currentID = instanceID
	log.Printf("Switched to S3 instance: %s (%s)", instanceID, m.instances[instanceID].Name)
	return nil
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
