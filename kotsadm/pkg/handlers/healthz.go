package handlers

import (
	"net/http"
)

type HealthzResponse struct {
	Version string         `json:"version"`
	GitSHA  string         `json:"gitSha"`
	Status  StatusResponse `json:"status"`
}

type StatusResponse struct {
	Database DatabaseResponse `json:"database"`
	Storage  StorageResponse  `json:"storage"`
}

type DatabaseResponse struct {
	Connected bool `json:"connected"`
}

type StorageResponse struct {
	Available bool `json:"available"`
}

// Healthz route is UNAUTHENTICATED
func Healthz(w http.ResponseWriter, r *http.Request) {
	// TODO
	isDatabaseConnected := true
	isStorageAvailable := true

	healthzResponse := HealthzResponse{
		Version: "test",
		GitSHA:  "test",
		Status: StatusResponse{
			Database: DatabaseResponse{
				Connected: isDatabaseConnected,
			},
			Storage: StorageResponse{
				Available: isStorageAvailable,
			},
		},
	}

	statusCode := 200
	if !isDatabaseConnected || !isStorageAvailable {
		statusCode = 419
	}

	JSON(w, statusCode, healthzResponse)
}
