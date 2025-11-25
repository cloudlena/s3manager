package s3manager

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
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
		current := manager.GetCurrentInstance()
		
		response := struct {
			Instances []S3InstanceInfo `json:"instances"`
			Current   string           `json:"current"`
		}{
			Instances: make([]S3InstanceInfo, len(instances)),
			Current:   current.ID,
		}
		
		for i, instance := range instances {
			response.Instances[i] = S3InstanceInfo{
				ID:   instance.ID,
				Name: instance.Name,
			}
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// HandleSwitchS3Instance switches to a specific S3 instance
func HandleSwitchS3Instance(manager *MultiS3Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		instanceID := vars["instanceId"]
		
		if instanceID == "" {
			http.Error(w, "instance ID is required", http.StatusBadRequest)
			return
		}
		
		err := manager.SetCurrentInstance(instanceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		
		current := manager.GetCurrentInstance()
		response := S3InstanceInfo{
			ID:   current.ID,
			Name: current.Name,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}