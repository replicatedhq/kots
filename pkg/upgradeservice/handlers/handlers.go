package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/logger"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ UpgradeServiceHandler = (*Handler)(nil)

type Handler struct {
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	velerov1.AddToScheme(scheme.Scheme)
}

func RegisterAPIRoutes(r *mux.Router, handler UpgradeServiceHandler) {
	// CAUTION: modifying this prefix WILL break backwards compatibility
	subRouter := r.PathPrefix("/api/v1/upgrade-service/app/{appSlug}").Subrouter()
	subRouter.Use(LoggingMiddleware, AppSlugMiddleware)

	subRouter.Path("").Methods("GET").HandlerFunc(handler.Info)
	subRouter.Path("/ping").Methods("GET").HandlerFunc(handler.Ping)

	subRouter.Path("/config").Methods("GET").HandlerFunc(handler.CurrentConfig)
	subRouter.Path("/liveconfig").Methods("POST").HandlerFunc(handler.LiveConfig)
	subRouter.Path("/config").Methods("PUT").HandlerFunc(handler.SaveConfig)
	subRouter.Path("/config/{filename}/download").Methods("GET").HandlerFunc(handler.DownloadFileFromConfig)

	subRouter.Path("/preflight/run").Methods("POST").HandlerFunc(handler.StartPreflightChecks)
	subRouter.Path("/preflight/result").Methods("GET").HandlerFunc(handler.GetPreflightResult)

	subRouter.Path("/deploy").Methods("POST").HandlerFunc(handler.Deploy)
}

func JSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
