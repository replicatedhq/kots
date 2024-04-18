package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/image"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/imageutil"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	orasretry "oras.land/oras-go/v2/registry/remote/retry"
)

type GetEmbeddedClusterRolesResponse struct {
	Roles []string `json:"roles"`
}

func (h *Handler) GetEmbeddedClusterNodes(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodes, err := embeddedcluster.GetNodes(r.Context(), client)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, nodes)
}

func (h *Handler) GetEmbeddedClusterNode(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	client, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nodeName := mux.Vars(r)["nodeName"]
	node, err := embeddedcluster.GetNode(r.Context(), client, nodeName)
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, node)
}

func (h *Handler) GetEmbeddedClusterRoles(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	roles, err := embeddedcluster.GetRoles(r.Context())
	if err != nil {
		logger.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	JSON(w, http.StatusOK, GetEmbeddedClusterRolesResponse{Roles: roles})
}

func (h *Handler) GetEmbeddedClusterArtifact(w http.ResponseWriter, r *http.Request) {
	if !util.IsEmbeddedCluster() {
		logger.Errorf("not an embedded cluster")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	apps, err := store.GetStore().ListInstalledApps()
	if err != nil {
		logger.Error(fmt.Errorf("failed to list installed apps: %w", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(apps) == 0 {
		logger.Error(fmt.Errorf("no installed apps found"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	a := apps[0]

	downstreams, err := store.GetStore().ListDownstreamsForApp(a.ID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to list downstreams for app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(downstreams) == 0 {
		logger.Error(errors.New("no downstreams found for app"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	d := downstreams[0]

	currentVersion, err := store.GetStore().GetCurrentDownstreamVersion(a.ID, d.ClusterID)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get current downstream version"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if currentVersion == nil {
		logger.Error(errors.New("no current downstream version found"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installation := currentVersion.KOTSKinds.Installation
	artifactType := mux.Vars(r)["artifactType"]
	artifactName := ""

	switch artifactType {
	case "charts":
		artifactName = installation.Spec.EmbeddedClusterArtifacts.Charts
	case "images-amd64":
		artifactName = installation.Spec.EmbeddedClusterArtifacts.ImagesAmd64
	case "binary-amd64":
		artifactName = installation.Spec.EmbeddedClusterArtifacts.BinaryAmd64
	case "metadata":
		artifactName = installation.Spec.EmbeddedClusterArtifacts.Metadata
	default:
		logger.Errorf("unknown artifact type %q", artifactType)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get k8s clientset"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	registryConfig, err := kotsadm.GetRegistryConfigFromCluster(util.PodNamespace, clientset)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get registry config from cluster"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ociArtifactPath := imageutil.NewEmbeddedClusterOCIArtifactPath(artifactName, imageutil.EmbeddedClusterArtifactOCIPathOptions{
		RegistryHost:      registryConfig.OverrideRegistry,
		RegistryNamespace: registryConfig.OverrideNamespace,
		ChannelID:         installation.Spec.ChannelID,
		UpdateCursor:      installation.Spec.UpdateCursor,
		VersionLabel:      installation.Spec.VersionLabel,
	})

	artifactReader, err := image.PullOCIArtifact(imagetypes.PullOCIArtifactOptions{
		Registry: dockerregistrytypes.RegistryOptions{
			Endpoint:  registryConfig.OverrideRegistry,
			Namespace: registryConfig.OverrideNamespace,
			Username:  registryConfig.Username,
			Password:  registryConfig.Password,
		},
		Repository: ociArtifactPath.Repository,
		Tag:        ociArtifactPath.Tag,
		HTTPClient: orasretry.DefaultClient,
	})
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to pull oci artifact"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer artifactReader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, artifactReader)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to copy oci artifact to response"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
