package handlers

import "net/http"

type UpgradeServiceHandler interface {
	Info(w http.ResponseWriter, r *http.Request)
	Ping(w http.ResponseWriter, r *http.Request)

	CurrentConfig(w http.ResponseWriter, r *http.Request)
	LiveConfig(w http.ResponseWriter, r *http.Request)
	SaveConfig(w http.ResponseWriter, r *http.Request)
	DownloadFileFromConfig(w http.ResponseWriter, r *http.Request)

	StartPreflightChecks(w http.ResponseWriter, r *http.Request)
	GetPreflightResult(w http.ResponseWriter, r *http.Request)

	Deploy(w http.ResponseWriter, r *http.Request)
}
