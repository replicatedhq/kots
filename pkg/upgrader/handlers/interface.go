package handlers

import "net/http"

type UpgraderHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)

	CurrentAppConfig(w http.ResponseWriter, r *http.Request)
	LiveAppConfig(w http.ResponseWriter, r *http.Request)
}
