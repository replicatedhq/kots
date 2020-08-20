package handlers

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	appstatustypes "github.com/replicatedhq/kots/kotsadm/pkg/appstatus/types"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
)

type GetAppDashboardResponse struct {
	AppStatus         *appstatustypes.AppStatus `json:"appStatus"`
	Metrics           []version.MetricChart     `json:"metrics"`
	PrometheusAddress string                    `json:"prometheusAddress"`
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

	a, err := store.GetStore().GetAppFromSlug(appSlug)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	appStatus, err := store.GetStore().GetAppStatus(a.ID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	parentSequence, err := downstream.GetCurrentParentSequence(a.ID, clusterID)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}

	prometheusAddress, err := store.GetStore().GetPrometheusAddress()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(500)
		return
	}
	if prometheusAddress == "" {
		prometheusAddress = os.Getenv("PROMETHEUS_ADDRESS")
	}

	metrics, err := version.GetMetricCharts(a.ID, parentSequence, prometheusAddress)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get metric charts"))
		metrics = []version.MetricChart{}
	}

	getAppDashboardResponse := GetAppDashboardResponse{
		AppStatus:         appStatus,
		Metrics:           metrics,
		PrometheusAddress: prometheusAddress,
	}

	JSON(w, 200, getAppDashboardResponse)
}
