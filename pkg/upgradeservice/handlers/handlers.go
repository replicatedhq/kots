package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/kots/pkg/logger"
	kotsscheme "github.com/replicatedhq/kotskinds/client/kotsclientset/scheme"
	troubleshootscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ UpgradeServiceHandler = (*Handler)(nil)

type Handler struct {
}

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
	troubleshootscheme.AddToScheme(scheme.Scheme)
	veleroscheme.AddToScheme(scheme.Scheme)
}

func RegisterRoutes(r *mux.Router, handler UpgradeServiceHandler) {
	// CAUTION: modifying this prefix WILL break backwards compatibility
	subRouter := r.PathPrefix("/api/v1/upgrade-service").Subrouter()
	subRouter.Use(LoggingMiddleware)

	subRouter.Path("/ping").Methods("GET").HandlerFunc(handler.Ping)

	subRouter.Path("/app/{appSlug}/config").Methods("GET").HandlerFunc(handler.CurrentAppConfig)
	subRouter.Path("/app/{appSlug}/liveconfig").Methods("POST").HandlerFunc(handler.LiveAppConfig)
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
