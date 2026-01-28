package s3manager

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// S3InstanceInfo represents the information about an S3 instance for API responses
type S3InstanceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HandleGetS3Instances returns all available S3 instances
func HandleGetS3Instances(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instances := manager.GetAllInstances()

		response := struct {
			Instances []S3InstanceInfo `json:"instances"`
		}{
			Instances: make([]S3InstanceInfo, len(instances)),
		}

		for i, instance := range instances {
			response.Instances[i] = S3InstanceInfo{
				ID:   instance.ID,
				Name: instance.Name,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			handleHTTPError(w, fmt.Errorf("error encoding JSON: %w", err))
			return
		}
	}
}
