package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmconfig "github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	upgradepreflight "github.com/replicatedhq/kots/pkg/upgradeservice/preflight"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeployAppRequest struct {
	IsSkipPreflights             bool `json:"isSkipPreflights"`
	ContinueWithFailedPreflights bool `json:"continueWithFailedPreflights"`
}

type DeployAppResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) DeployApp(w http.ResponseWriter, r *http.Request) {
	response := DeployAppResponse{
		Success: false,
	}

	params := GetContextParams(r)

	request := DeployAppRequest{}
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

	canDeploy, reason, err := canDeployApp(params, kotsKinds)
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

	tgzArchiveKey := fmt.Sprintf("pending-versions/%s/%s-%s.tar.gz", params.AppSlug, params.UpdateChannelID, params.UpdateCursor)
	if err := apparchive.CreateAppVersionArchive(params.AppArchive, tgzArchiveKey); err != nil {
		response.Error = "failed to create app version archive"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	ecInstallationName, err := embeddedcluster.MaybeStartClusterUpgrade(r.Context(), kotsKinds)
	if err != nil {
		response.Error = "failed to start cluster upgrade"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	if err := createPendingDeploymentCM(r.Context(), request, params, tgzArchiveKey, kotsKinds, ecInstallationName); err != nil {
		response.Error = "failed to create app version configmap"
		logger.Error(errors.Wrap(err, response.Error))
		JSON(w, http.StatusInternalServerError, response)
		return
	}

	// TODO NOW: preflight reporting?

	response.Success = true
	JSON(w, http.StatusOK, response)
}

func canDeployApp(params types.UpgradeServiceParams, kotsKinds *kotsutil.KotsKinds) (bool, string, error) {
	registrySettings := registrytypes.RegistrySettings{
		Hostname:   params.RegistryEndpoint,
		Username:   params.RegistryUsername,
		Password:   params.RegistryPassword,
		Namespace:  params.RegistryNamespace,
		IsReadOnly: params.RegistryIsReadOnly,
	}

	needsConfig, err := kotsadmconfig.NeedsConfiguration(params.AppSlug, params.NextSequence, params.AppIsAirgap, kotsKinds, registrySettings)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to check if version needs configuration")
	}
	if needsConfig {
		return false, "cannot deploy because version needs configuration", nil
	}

	pd, err := upgradepreflight.GetPreflightData()
	if err != nil {
		return false, "", errors.Wrap(err, "failed to get preflight data")
	}
	if pd.Result != nil && pd.Result.HasFailingStrictPreflights {
		return false, "cannot deploy because a strict preflight check has failed", nil
	}

	return true, "", nil
}

// createPendingDeploymentCM creates a configmap with the app version info which
// gets detected by the operator of the new kots version to deploy the app
func createPendingDeploymentCM(
	ctx context.Context,
	request DeployAppRequest,
	params types.UpgradeServiceParams,
	tgzArchiveKey string,
	kotsKinds *kotsutil.KotsKinds,
	ecInstallationName string,
) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	source := "Upstream Update"
	if params.AppIsAirgap {
		source = "Airgap Update"
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("kotsadm-%s-pending-deployment", params.AppSlug),
			Labels: map[string]string{
				// exclude from backup so this app version is not deployed on restore
				kotsadmtypes.ExcludeKey:      kotsadmtypes.ExcludeValue,
				"kots.io/pending-deployment": "true",
			},
		},
		Data: map[string]string{
			"app-id":                          params.AppID,
			"app-slug":                        params.AppSlug,
			"app-version-archive":             tgzArchiveKey,
			"base-sequence":                   fmt.Sprintf("%d", params.BaseSequence),
			"version-label":                   params.UpdateVersionLabel,
			"source":                          source,
			"is-airgap":                       fmt.Sprintf("%t", params.AppIsAirgap),
			"channel-id":                      params.UpdateChannelID,
			"update-cursor":                   params.UpdateCursor,
			"skip-preflights":                 fmt.Sprintf("%t", request.IsSkipPreflights),
			"continue-with-failed-preflights": fmt.Sprintf("%t", request.ContinueWithFailedPreflights),
			"kots-version":                    params.UpdateKOTSVersion,
			"ec-installation-name":            ecInstallationName,
		},
	}

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing configmap")
		}
		_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create configmap")
		}
		return nil
	}

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update configmap")
	}

	return nil
}
