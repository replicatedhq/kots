package handlers

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/appstatus"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/kotsadmparams"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

type GetAppDashboardResponse struct {
	AppStatus         *appstatus.AppStatus  `json:"appStatus"`
	Metrics           []version.MetricChart `json:"metrics"`
	PrometheusAddress string                `json:"prometheusAddress"`
}

func GetAppDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := requireValidSession(w, r); err != nil {
		logger.Error(err)
		return
	}

	appSlug := mux.Vars(r)["appSlug"]
	clusterID := mux.Vars(r)["clusterId"]

	a, err := app.GetFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	appStatus, err := appstatus.Get(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	parentSequence, err := downstream.GetDownstreamParentSequence(a.ID, clusterID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	metrics, err := version.GetMetricCharts(a.ID, parentSequence)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	prometheusAddress, err := kotsadmparams.Get("PROMETHEUS_ADDRESS")
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	if prometheusAddress == "" {
		prometheusAddress = os.Getenv("PROMETHEUS_ADDRESS")
	}

	getAppDashboardResponse := GetAppDashboardResponse{
		AppStatus:         appStatus,
		Metrics:           metrics,
		PrometheusAddress: prometheusAddress,
	}

	JSON(w, 200, getAppDashboardResponse)
}
