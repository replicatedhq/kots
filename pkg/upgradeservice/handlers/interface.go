package handlers

import "net/http"

type UpgradeServiceHandler interface {
	GetApp(w http.ResponseWriter, r *http.Request)
	Ping(w http.ResponseWriter, r *http.Request)

	CurrentAppConfig(w http.ResponseWriter, r *http.Request)
	LiveAppConfig(w http.ResponseWriter, r *http.Request)
	SaveAppConfig(w http.ResponseWriter, r *http.Request)
	DownloadFileFromConfig(w http.ResponseWriter, r *http.Request)

	StartPreflightChecks(w http.ResponseWriter, r *http.Request)
	GetPreflightResult(w http.ResponseWriter, r *http.Request)

	DeployApp(w http.ResponseWriter, r *http.Request)
}
