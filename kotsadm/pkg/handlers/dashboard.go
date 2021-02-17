package handlers

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	appstatustypes "github.com/replicatedhq/kots/pkg/api/appstatus/types"
	"github.com/replicatedhq/kots/pkg/logger"
)

type GetAppDashboardResponse struct {
	AppStatus         *appstatustypes.AppStatus `json:"appStatus"`
	Metrics           []version.MetricChart     `json:"metrics"`
	PrometheusAddress string                    `json:"prometheusAddress"`
}

func (h *Handler) GetAppDashboard(w http.ResponseWriter, r *http.Request) {
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
