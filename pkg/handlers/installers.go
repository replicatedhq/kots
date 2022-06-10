package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-test/deep"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/version"
	kurlclientset "github.com/replicatedhq/kurl/kurlkinds/client/kurlclientset"
	kurlkinds "github.com/replicatedhq/kurl/kurlkinds/pkg/apis/cluster/v1beta1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CheckInstallerResponse struct {
	Success      bool     `json:"success"`
	HasInstaller bool     `json:"hasInstaller"`
	Diff         []string `json:"diff,omitempty"`
	Error        string   `json:"error,omitempty"`
}

func (h *Handler) CheckInstaller(w http.ResponseWriter, r *http.Request) {
	checkInstallerVersionResponse := CheckInstallerResponse{
		Success:      false,
		HasInstaller: false,
	}

	appSlug := mux.Vars(r)["appSlug"]

	sequence, err := strconv.ParseInt(mux.Vars(r)["sequence"], 10, 64)
	if err != nil {
		errMsg := "failed to parse sequence number"
		logger.Error(errors.Wrap(err, errMsg))
		checkInstallerVersionResponse.Error = errMsg
		JSON(w, http.StatusBadRequest, checkInstallerVersionResponse)
		return
	}

	archiveFiles, err := version.GetAppVersionArchiveFiles(appSlug, int64(sequence))
	if err != nil {
		errMsg := fmt.Sprintf("failed to get app archive files for app %s sequence %d", appSlug, sequence)
		logger.Error(errors.Wrap(err, errMsg))
		checkInstallerVersionResponse.Error = errMsg
		JSON(w, http.StatusInternalServerError, checkInstallerVersionResponse)
		return
	}

	installerYAML := getUpstreamInstallerYAML(archiveFiles)
	if installerYAML != "" {
		checkInstallerVersionResponse.HasInstaller = true

		resolvedInstallerJSON, err := getResolvedInstallerAnnotation(installerYAML)
		if err != nil {
			errMsg := "failed to get resolved installer annotation"
			logger.Error(errors.Wrap(err, errMsg))
			checkInstallerVersionResponse.Error = errMsg
			JSON(w, http.StatusInternalServerError, checkInstallerVersionResponse)
			return
		}

		releaseInstaller := &kurlkinds.Installer{}
		err = json.Unmarshal([]byte(resolvedInstallerJSON), releaseInstaller)
		if err != nil {
			errMsg := "failed to unmarshal resolved installer"
			logger.Error(errors.Wrap(err, errMsg))
			checkInstallerVersionResponse.Error = errMsg
			JSON(w, http.StatusInternalServerError, checkInstallerVersionResponse)
			return
		}

		deployedInstaller, err := getDeployedInstaller()
		if err != nil {
			errMsg := "failed to get currently deployed installer"
			logger.Error(errors.Wrap(err, errMsg))
			checkInstallerVersionResponse.Error = errMsg
			JSON(w, http.StatusInternalServerError, checkInstallerVersionResponse)
			return
		}

		logger.Infof("comparing %s sequence %d installer to deployed installer %s", appSlug, sequence, deployedInstaller.Name)

		diff := compareInstallers(deployedInstaller, releaseInstaller)
		checkInstallerVersionResponse.Diff = diff
	}

	checkInstallerVersionResponse.Success = true
	JSON(w, http.StatusOK, checkInstallerVersionResponse)
}

func getUpstreamInstallerYAML(archiveFiles map[string][]byte) string {
	for file, bytes := range archiveFiles {
		if strings.HasPrefix(file, "/upstream/") {
			doc := map[string]interface{}{}
			if err := yaml.Unmarshal(bytes, &doc); err != nil {
				logger.Error(errors.Wrapf(err, "failed to unmarshal %s", file))
			}
			if doc["apiVersion"] == "kurl.sh/v1beta1" && doc["kind"] == "Installer" {
				return string(bytes)
			}
			if doc["apiVersion"] == "cluster.kurl.sh/v1beta1" && doc["kind"] == "Installer" {
				return string(bytes)
			}
		}
	}
	return ""
}

func getResolvedInstallerAnnotation(installerYAML string) (string, error) {
	gvk := &base.OverlySimpleGVK{}
	err := yaml.Unmarshal([]byte(installerYAML), gvk)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal installer yaml")
	}

	resolvedInstallerAnnotation := gvk.Metadata.Annotations["kots.io/resolved-installer"]
	resolvedInstallerJSON, ok := resolvedInstallerAnnotation.(string)
	if !ok {
		return "", errors.New("failed to parse resolved installer annotation")
	}

	return resolvedInstallerJSON, nil
}

func compareInstallers(deployedInstaller, releaseInstaller *kurlkinds.Installer) []string {
	if releaseInstaller.Spec.Kurl.AdditionalNoProxyAddresses == nil {
		// if this is nil, it will be set to an empty string slice by kurl, so lets do so before comparing
		releaseInstaller.Spec.Kurl.AdditionalNoProxyAddresses = []string{}
	}

	if releaseInstaller.Spec.Kotsadm.ApplicationSlug == "" {
		// application slug may be injected into the installer, so remove it if not specified in release installer
		deployedInstaller.Spec.Kotsadm.ApplicationSlug = ""
	}
	// NOTE: The above may need to be done for applicationVerionLabel, but it's not a part of kurlkinds yet

	diff := deep.Equal(deployedInstaller.Spec, releaseInstaller.Spec)

	return diff
}

func getDeployedInstaller() (*kurlkinds.Installer, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	cm, err := clientset.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kurl-config", v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kurl-config")
	}

	installerId := cm.Data["installer_id"]

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	kurlClientset, err := kurlclientset.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kurl clientset")
	}

	// QUESTION: Will the kurl installers always be deployed to the default namespace?
	deployedInstaller, err := kurlClientset.ClusterV1beta1().Installers("default").Get(context.TODO(), installerId, v1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get installer %s", installerId)
	}

	return deployedInstaller, nil
}
