package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/upgradeservice/deploy"
)

type DeployEC2Request struct {
	IsSkipPreflights             bool `json:"isSkipPreflights"`
	ContinueWithFailedPreflights bool `json:"continueWithFailedPreflights"`
}

type DeployEC2Response struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DeployEC2(w http.ResponseWriter, r *http.Request) {
	response := DeployEC2Response{
		Success: false,
	}

	params := GetContextParams(r)

	request := DeployEC2Request{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response.Error = "failed to decode request body"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	kotsKinds, err := kotsutil.LoadKotsKinds(params.AppArchive)
	if err != nil {
		response.Error = "failed to load kots kinds from path"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	canDeploy, reason, err := deploy.CanDeployEC2(deploy.CanDeployEC2Options{
		Params:           params,
		KotsKinds:        kotsKinds,
		RegistrySettings: registrySettings,
	})
	if err != nil {
		response.Error = "failed to check if app can be deployed"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}
	if !canDeploy {
		response.Error = reason
		logger.Error(errors.New(response.Error))
		JSON(w, http.StatusBadRequest, response)
		return
	}

	if err := deploy.DeployEC2(deploy.DeployEC2Options{
		Ctx:                          r.Context(),
		IsSkipPreflights:             request.IsSkipPreflights,
		ContinueWithFailedPreflights: request.ContinueWithFailedPreflights,
		Params:                       params,
		KotsKinds:                    kotsKinds,
		RegistrySettings:             registrySettings,
	}); err != nil {
		response.Error = "failed to deploy app"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	response.Success = true
	JSON(w, http.StatusOK, response)
}
