package handlers

import (
	"net/http"

	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"
	versiontypes "github.com/replicatedhq/kots/pkg/api/version/types"
	"github.com/replicatedhq/kots/pkg/logger"
)

type GetApplicationPortsResponse struct {
	Ports []versiontypes.ForwardedPort `json:"ports"`
}

// NOTE: this uses special kots token authorization
func (h *Handler) GetApplicationPorts(w http.ResponseWriter, r *http.Request) {
	if err := requireValidKOTSToken(w, r); err != nil {
		logger.Error(err)
		return
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetApplicationPortsResponse{}

	for _, app := range apps {
		ports, err := version.GetForwardedPortsFromAppSpec(app.ID, app.CurrentSequence)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response.Ports = append(response.Ports, ports...)
	}

	JSON(w, 200, response)
}
