package handlers

import "net/http"

type UpgradeServiceHandler interface {
	Ping(w http.ResponseWriter, r *http.Request)

	CurrentAppConfig(w http.ResponseWriter, r *http.Request)
	LiveAppConfig(w http.ResponseWriter, r *http.Request)
	SaveAppConfig(w http.ResponseWriter, r *http.Request)
	DownloadFileFromConfig(w http.ResponseWriter, r *http.Request)

	GetPreflightResult(w http.ResponseWriter, r *http.Request)
}
