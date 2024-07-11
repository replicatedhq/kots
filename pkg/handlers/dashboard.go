package handlers

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
)

type GetAppDashboardResponse struct {
	AppStatus            *appstatetypes.AppStatus `json:"appStatus"`
	Metrics              []version.MetricChart    `json:"metrics"`
	PrometheusAddress    string                   `json:"prometheusAddress"`
	EmbeddedClusterState string                   `json:"embeddedClusterState"`
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

	parentSequence, err := store.GetStore().GetCurrentParentSequence(a.ID, clusterID)
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

	metrics := []version.MetricChart{}
	if prometheusAddress != "" {
		graphs, err := version.GetGraphs(a, parentSequence, store.GetStore())
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to get graphs for app %s sequence %d. falling back to default graphs", a.Slug, parentSequence))
		}
		metrics = version.GetMetricCharts(graphs, prometheusAddress)
	}

	embeddedClusterState := ""
	if util.IsEmbeddedCluster() {
		kbClient, err := k8sutil.GetKubeClient(r.Context())
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}
		ecInstallation, err := embeddedcluster.GetCurrentInstallation(r.Context(), kbClient)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(500)
			return
		}
		embeddedClusterState = ecInstallation.Status.State
	}

	getAppDashboardResponse := GetAppDashboardResponse{
		AppStatus:            appStatus,
		Metrics:              metrics,
		PrometheusAddress:    prometheusAddress,
		EmbeddedClusterState: embeddedClusterState,
	}

	JSON(w, 200, getAppDashboardResponse)
}
